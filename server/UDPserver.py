import os
import socket
import threading
import logging
import inspect
import time
import Event as event
EventHandler = event.EventHandler

from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend

logging.basicConfig(level=logging.DEBUG)


class Client:
    def __init__(self, on_remove, host, encryption=False, aes_key=5, aes_initkey=5):
        self._on_remove = on_remove
        self._host = host
        self._port = None
        self._encryption = encryption
        self._aes_key = aes_key
        self._aes_initkey = aes_initkey
        self._message_callback = EventHandler()

    def address(self):
        """Return the server's address."""
        return (self._host, self._port)

    def stop(self):
        logging.debug(f"Removing UDP client {self._host} from whitelist.")
        self._message_callback = EventHandler()
        self._on_remove([self._host, self._port])

    def on_event(self, event_type, callback):
        # Get the number of parameters the callback has
        num_params = len(inspect.signature(callback).parameters)

        if event_type == "message":
            if num_params == 2:
                return self._message_callback.add_event(callback)
            else:
                logging.error(f"Invalid number of parameters for 'on_message' event. Expected 2, got {num_params}.")
        else:
            logging.warning(f"Unsupported event type: {event_type}")

    def remove_event(self, event_type, callback):
        if event_type == "message":
            self._message_callback.remove_event(callback)
        else:
            logging.warning(f"Unsupported event type: {event_type}")


class Server:
    def __init__(self, host, port, encryption=False, buffer_size=1024):
        logging.debug("Initializing UDP server.")
        self._host = host
        self._port = port
        self._running = False
        self._socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self._encryption = encryption
        self._main_thread = None
        self._clients = dict()      # {host: [Client, ...]}
        self._clients_lock = threading.Lock()
        self._connected_callbacks = EventHandler()
        self._buffer_size = buffer_size

    def start(self):
        """Start the server."""
        if self._running:
            logging.warning("UDP server is already running.")
            return

        logging.debug("Starting UDP server.")
        self._running = True
        self._socket.bind((self._host, self._port))

        self._main_thread = threading.Thread(target=self._listen)
        self._main_thread.start()

        logging.info(f"Server started at {self._host}:{self._port}")

    def stop(self):
        """Stop the server."""
        if not self._running:
            logging.warning("UDP server is not running.")
            return
        logging.debug("Stopping UDP server.")
        self._running = False
        self._socket.sendto(b"exit", (self._host, self._port))
        self._socket.close()

        # stop all clients
        tempclients = None
        with self._clients_lock:
            tempclients = self._clients.values().copy()
        for clientslist in tempclients:
            for client in clientslist:
                client.stop()

        self._main_thread.join()
        logging.info("UDP server stopped.")




    def add_client(self, host, aes_key=5, aes_initkey=5):
        """Add a client to the whitelist."""
        logging.debug(f"Adding UDP client {host} to whitelist.")
        udp_encryption = False
        if self._encryption:
            udp_encryption = True

        client = Client(self._remove_client, host, udp_encryption, aes_key, aes_initkey)

        with self._clients_lock:
            if host in self._clients:
                self._clients[host].append(client)
            else:
                self._clients[host] = [client]
            return client

    def remove_client(self, address):
        """Remove a client from the whitelist."""
        logging.debug(f"Removing UDP client {address}.")
        host = address[0]
        with self._clients_lock:
            clientslist = self._clients.get(host)
            if clientslist is None:
                logging.debug(f"UDP client {address} not found.")
                return
            for client in clientslist:
                if client.address() == address:
                    client.stop()
                    return
            logging.debug(f"UDP client {address} not found in list.")

    def _remove_client(self, address):
        """Remove a client from the whitelist. Internal use only."""
        logging.debug(f"Removing UDP client {address} from whitelist.")
        host = address[0]
        with self._clients_lock:
            clientslist = self._clients.get(host)
            if clientslist is None:
                logging.debug(f"UDP client {address} not found.")
                return
            for client in clientslist:
                if client.address() == address:
                    clientslist.remove(client)
                    break
                
            if len(clientslist) == 0:
                self._clients.pop(host)



    def on_connected(self, callback):
        """Register a callback for when a client connects."""
        # Get the number of parameters the callback has
        num_params = len(inspect.signature(callback).parameters)

        if num_params == 1:
            return self._connected_callbacks.add_event(callback)
        else:
            logging.error(f"Invalid number of parameters for 'on_connected' event. Expected 1, got {num_params}.")

    def remove_on_connected(self, callback):
        self._connected_callbacks.remove_event(callback)




    def _decrypt(self, encrypted_data, aes_key, aes_initkey):
        """Decrypt the received data."""
        logging.debug(f"Decrypting data")

        cipher = Cipher(algorithms.AES(aes_key), modes.CFB(aes_initkey), backend=default_backend())

        decryptor = cipher.decryptor()

        plaintext = decryptor.update(encrypted_data) + decryptor.finalize()
        return plaintext



    def _handle_socket_errors(self, error):
        """Centralize error handling for socket-related errors."""
        logging.debug(f"UDP socket error: {error}")
        self.stop()

    def _listen(self):
        """Listen for incoming messages."""
        logging.debug("Listening for incoming UDP messages.")
        while self._running:
            try:
                data, address = self._socket.recvfrom(self._buffer_size)
                host = address[0]
                port = address[1]
            except socket.error as e:
                self._handle_socket_errors(e)
                break

            clientlist = None
            with self._clients_lock:
                clientlist = self._clients.get(host)
            if clientlist is None:
                logging.debug(f"Received UDP message from {address}, which is not in the whitelist/clientlist.")
                continue

            client = None
            for c in clientlist:
                if c.address() == address:
                    client = c
                    break

            if client is None:
                 for c in clientlist:
                     if c._port is None:
                         client = c
                         client._port = port
                         break
                     
            if client is None:
                logging.debug(f"Received UDP message from {address}, which is not in the whitelist/clientlist.")
                continue

            if client._port is None:
                client._port = port
                self._connected_callbacks.emit(client)


            if self._encryption:
                data = self._decrypt(data, client._aes_key, client._aes_initkey)

            logging.debug(f"Received message from {address}: {data}")
            client._message_callback.emit(client, data)



def main():
    srv = Server("127.0.0.1", 5001, True)

    srv.on_connected(lambda client: print(f"Client {client.address()} connected."))

    aes_key = os.urandom(32)  # Generate a random 32 bytes AES key
    aes_initkey = os.urandom(16) # Generate a random 16 bytes AES init key
    print(f"AES Key: {aes_key}")
    print(f"AES Init Key: {aes_initkey}")
    c1 = srv.add_client("127.0.0.1", aes_key, aes_initkey)

    c1.on_event("message", lambda c, data: print(f"Client {c.address()}: {data}"))

    srv.start()

if __name__ == "__main__":
    main()