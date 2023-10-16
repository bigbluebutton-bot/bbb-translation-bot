import socket
import threading
import logging
import time
import inspect
from concurrent.futures import ThreadPoolExecutor

from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa, padding
from cryptography.hazmat.primitives import hashes
import os


BUFFER_SIZE = 1024
logging.basicConfig(level=logging.DEBUG)


# Convert the byte string to a list of integers for logging
def byte_string_to_int_list(byte_string):
    return [byte for byte in byte_string]

class EventHandler:
    """Class responsible for managing callbacks."""

    def __init__(self):
        logging.debug("Initializing EventHandler.")
        self._callbacks = {}
        self._event_lock = threading.Lock()

    def add_event(self, callback):
        """Add a new event callback and return its unique ID."""
        with self._event_lock:
            event_id = callback
            logging.debug(f"Adding event with ID: {event_id}")
            self._callbacks[event_id] = callback
            return event_id

    def remove_event(self, event_id):
        """Remove an event callback using its ID."""
        logging.debug(f"Removing event with ID: {event_id}")
        with self._event_lock:
            self._callbacks.pop(event_id, None)

    def emit(self, *args):
        """Trigger all the registered callbacks with the provided arguments."""
        threads = []  # To keep track of the threads

        with self._event_lock:
            for event_id, callback in self._callbacks.items():
                logging.debug(f"Emitting event with ID: {event_id}")
                # Wrap the callback execution in a thread
                t = threading.Thread(target=callback, args=args)
                threads.append(t)
                t.start()

        # Wait for all threads to finish
        for t in threads:
            t.join()


class Server:
    """Class representing a TCP server."""

    def __init__(self, host, port, timeout=5, encryption=False, backlog=5, max_threads=10):
        logging.debug("Initializing Server.")
        self.host = host
        self.port = port
        self.backlog = backlog
        self.timeout = timeout
        self._connected_callbacks = EventHandler()
        self._clients = []
        self._clients_lock = threading.Lock()
        self._socket = None
        self._running = False
        self.max_threads = max_threads
        self.main_accept_clients_thread = None
        self._thread_pool = None  # Delay the initialization of the ThreadPoolExecutor
        self.active_clients_count = 0
        self._encryption = encryption
        self.public_key = None
        self.private_key = None

    def generate_keys(self):
        """Generate RSA keys."""
        logging.debug("Generating RSA keys.")
        self.private_key = rsa.generate_private_key(
            public_exponent=65537,
            key_size=self._encryption,
        )
        self.public_key = self.private_key.public_key()


    def start(self):
        """Start the server."""
        if self._encryption:
            self.generate_keys()

        logging.debug("Starting server.")
        if self._socket:
            logging.warning("Server already started.")
            return

        self._running = True

        self.main_accept_clients_thread = threading.Thread(target=self._accept_clients) # Create the thread here
        self._thread_pool = ThreadPoolExecutor(self.max_threads)  # Initialize a thread pool when server starts

        try:
            self._socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self._socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            self._socket.settimeout(self.timeout)  
            self._socket.bind((self.host, self.port))
            self._socket.listen(self.backlog)

            logging.debug("Thread: Starting client acceptance.")
            self.main_accept_clients_thread.start()

        except Exception as e:
            logging.error(f"Failed to start server: {e}")
            self._running = False

        logging.info(f"Server started: {self.host}:{self.port}")

    def _accept_clients(self):
        """Private method to accept incoming clients."""
        logging.debug("Accepting clients.")
        while True:
            if not self._running:
                break
            
            try:
                logging.debug("Waiting for client...")

                conn, addr = self._socket.accept()
                if conn and self._running:
                    logging.debug(f"Accepted client: {addr}")
                    client = self._Client(self._remove_client, conn, addr, self.timeout, self._encryption, self.public_key, self.private_key)
                    with self._clients_lock:
                        self._clients.append(client)
                    self._connected_callbacks.emit(client)
                    logging.debug(f"Thread: Starting for client: {addr}")
                    self._thread_pool.submit(client.start)  # Use the thread pool to handle clients
                    self.active_clients_count += 1  # Increment counter here
            except socket.timeout:
                logging.debug("Main socket timeout. But can be ignored.")
                pass
            except socket.error as e:
                if e.errno == 10038:
                    logging.info("Server socket closed. Stopping client acceptance.")
                else:
                    logging.error(f"Error accepting clients: {e}")


    def _remove_client(self, client):
        """Private method to remove a client from the server's client list."""
        with self._clients_lock:
            logging.debug(f"Removing client: {client.addr}")
            self._clients.remove(client)
            self.active_clients_count -= 1  # Decrement counter here


    def stop(self):
        """Stop the server."""
        logging.debug("Stopping server.")

        self._running = False

        # 1. Close all client connections
        logging.debug("Stopping clients.")
        for client in self._clients[:]:
            client.stop()

        # 2. Close the server socket
        logging.debug("Stopping server socket.")
        if self._socket:
            self._socket.close()
            self._socket = None


        logging.debug("Thread: Stopping client acceptance.")
        # If needed, join the thread to wait for its completion
        if self.main_accept_clients_thread and self.main_accept_clients_thread.is_alive():
            self.main_accept_clients_thread.join()

        # 3. Shutdown the thread pool
        logging.debug("Thread: Shutting down thread pool.")
        if self._thread_pool:
            self._thread_pool.shutdown(wait=True)


    def on_connected(self, callback):
        """Register a callback for when a client connects."""
        logging.debug("Registering 'connected' event.")
        # Get the number of parameters the callback has
        num_params = len(inspect.signature(callback).parameters)

        if num_params != 1:
            logging.error(f"Invalid number of parameters for 'connected' event. Expected 1, got {num_params}.")
            return
        
        return self._connected_callbacks.add_event(callback)

    def remove_connected_event(self, event_id):
        """Remove the connected event callback using its ID."""
        logging.debug("Removing 'connected' event.")
        self._connected_callbacks.remove_event(event_id)

    class _Client:
        """Class representing a client."""

        def __init__(self, on_remove, conn, addr, timeout=5, encryption=False, public_key=None, private_key=None):
            logging.debug("Initializing Client.")
            self._on_remove = on_remove
            self.conn = conn
            self.conn.settimeout(timeout)
            self.addr = addr
            self._disconnected_callbacks = EventHandler()
            self._timeout_callbacks = EventHandler()
            self._message_callbacks = EventHandler()
            self._running = False
            self._encryption = encryption

            self.server_publickey = public_key
            self.server_privatekey = private_key

            self.client_key = None
            self.client_initkey = None

        def start(self):
            """Start the client listener."""
            logging.debug(f"Client[{self.addr}] Starting client.")

            if not self.conn or self._running:
                return
            self._running = True

            if self._encryption:
                self._send_server_publickey()
                self._listen_for_clientkey()

            self.send(b"OK")

            self._listen()


        def _listen_for_clientkey(self):
            """Listen for the client's key."""
            logging.debug(f"Client[{self.addr}] Listening for client key.")
            data = self.conn.recv(BUFFER_SIZE)
            if data:
                logging.debug(f"Client[{self.addr}] Received client key: {byte_string_to_int_list(data)}")
                init_and_key = self.server_privatekey.decrypt(
                    data,
                    padding.OAEP(
                        mgf=padding.MGF1(algorithm=hashes.SHA256()),
                        algorithm=hashes.SHA256(),
                        label=None
                    )
                )

                self.client_initkey = init_and_key[:16] # the first 16 bytes are the init vector
                self.client_key = init_and_key[16:]     # the rest is the key

                logging.debug(f"Client[{self.addr}] Decrypted AES Key: {byte_string_to_int_list(self.client_key)}")
                logging.debug(f"Client[{self.addr}] Decrypted AES IV: {byte_string_to_int_list(self.client_initkey)}")



            else:
                logging.debug(f"Client[{self.addr}] No data received. Closing connection.")
                self.stop()


        def _send_server_publickey(self):
            """Send the server's public key to the client."""
            logging.debug(f"Client[{self.addr}] Sending server public key.")

            # Get server_publickey
            server_publickey = self.server_publickey.public_bytes(
                encoding=serialization.Encoding.PEM,
                format=serialization.PublicFormat.SubjectPublicKeyInfo
            )

            # Send the encrypted AES key to the client
            self.send(server_publickey)


        def _decrypt(self, encrypted_data):
            """Decrypt the received data."""
            logging.debug(f"Client[{self.addr}] Decrypting data")
            # Erstelle ein Cipher-Objekt
            cipher = Cipher(algorithms.AES(self.client_key), modes.CFB(self.client_initkey), backend=default_backend())
            # Erstelle ein Decryptor-Objekt
            decryptor = cipher.decryptor()
            # Entschlüssle die Daten
            plaintext = decryptor.update(encrypted_data) + decryptor.finalize()
            return plaintext
        
        def _encrypt(self, data):
            """Encrypt the data to be sent."""
            logging.debug(f"Client[{self.addr}] Encrypting data: {data}")
            # Erstelle ein Cipher-Objekt
            cipher = Cipher(algorithms.AES(self.client_key), modes.CFB(self.client_initkey), backend=default_backend())
            # Erstelle ein Encryptor-Objekt
            encryptor = cipher.encryptor()
            # Verschlüssle die Daten
            ciphertext = encryptor.update(data) + encryptor.finalize()
            return ciphertext


        def _handle_socket_errors(self, error):
            """Centralize error handling for socket-related errors."""
            logging.debug(f"Client[{self.addr}] Socket error: {error}")
            self.stop()

        def _listen(self):
            """Private method to listen for incoming data from the client."""
            logging.debug(f"Client[{self.addr}] Listening for data.")
            while self._running:
                try:
                    # logging.debug(f"Client[{self.addr}] Waiting for data.")
                    data = self.conn.recv(BUFFER_SIZE)
                    if data:
                        # Decrypt the data if encryption is enabled
                        if self._encryption:
                            logging.debug(f"Client[{self.addr}] Received encrypted data: {byte_string_to_int_list(data)}")
                            data = self._decrypt(data)
                        
                        logging.debug(f"Client[{self.addr}] Received data: {byte_string_to_int_list(data)}")
                        self._message_callbacks.emit(self, data)

                except (socket.timeout, socket.error, OSError) as e:  # Merged the error handling
                    if isinstance(e, socket.timeout):
                        self._timeout_callbacks.emit(self)
                    self._handle_socket_errors(e)

        def stop(self):
            """Stop the client and close its connection."""
            logging.debug(f"Client[{self.addr}] Stopping client.")
            self._running = False

            if not self.conn:
                return
            self._disconnected_callbacks.emit(self)

            try:
                self.conn.shutdown(socket.SHUT_RDWR)
                self.conn.close()
            except Exception as e:
                logging.error(f"Error while closing client connection: {e}")
            self.conn = None
            self._on_remove(self)

            logging.debug(f"Thread: Stopped for client: {self.addr}")

        def send(self, data):
            """Send data to the client."""
            try:
                logging.debug(f"Client[{self.addr}] Sending data: {data}")

                # Encrypt the data if encryption is enabled
                if self._encryption and self.client_key and self.client_initkey:
                    data = self._encrypt(data)

                self.conn.sendall(data)
            except (OSError, Exception) as e:
                self._handle_socket_errors(e)

        # Simplified event handlers for the client
        def on_event(self, event_type, callback):
            """Register an event callback based on the event type."""
            
            # Get the number of parameters the callback has
            num_params = len(inspect.signature(callback).parameters)

            # Check if the number of parameters matches the expected value for each event type
            if event_type == "disconnected":
                if num_params == 1:
                    return self._disconnected_callbacks.add_event(callback)
                else:
                    logging.error(f"Invalid number of parameters for 'disconnected' event. Expected 1, got {num_params}.")
            elif event_type == "timeout":
                if num_params == 1:
                    return self._timeout_callbacks.add_event(callback)
                else:
                    logging.error(f"Invalid number of parameters for 'timeout' event. Expected 1, got {num_params}.")
            elif event_type == "message":
                if num_params == 2:
                    return self._message_callbacks.add_event(callback)
                else:
                    logging.error(f"Invalid number of parameters for 'message' event. Expected 2, got {num_params}.")
            else:
                logging.warning(f"Unsupported event type: {event_type}")


        def remove_event(self, event_type, event_id):
            """Remove an event callback based on the event type."""
            if event_type == "disconnected":
                self._disconnected_callbacks.remove_event(event_id)
            elif event_type == "timeout":
                self._timeout_callbacks.remove_event(event_id)
            elif event_type == "message":
                self._message_callbacks.remove_event(event_id)
            else:
                logging.warning(f"Unsupported event type: {event_type}")


# EXAMPLE USAGE
SECRET_TOKEN = "your_secret_token"

def validate_token(client, data):
    """Check if the received token matches the secret token."""
    print(data)
    if data.decode('utf-8') != SECRET_TOKEN:
        logging.warning(f"Invalid token from {client.addr}. Closing connection.")
        client.stop()
    else:
        logging.info(f"Valid token received from {client.addr}. Connection authorized.")
        client.remove_event("message", validate_token)
        client.on_event("message", handle_client_message)

def handle_client_message(client, data):
    """Handle received message after token validation."""
    logging.info(f"Received from {client.addr}: {data.decode('utf-8')}")  # Decode data for logging
    client.send(b"OK")

def on_connected(client):
    """Handle new client connection."""
    logging.info(f"Connected by {client.addr}")
    client.on_event("disconnected", lambda c: logging.info(f"Disconnected by {client.addr}"))
    client.on_event("timeout", lambda c: logging.info(f"Timeout by {client.addr}"))
    client.on_event("message", validate_token)

def main():
    srv = Server('localhost', 5000, 5, 4096, 5, 10)

    srv.on_connected(on_connected)

    logging.info("Starting server: 127.0.0.1:5000...")
    srv.start()
    logging.info("Waiting for connections...")

    try:
        while True:  # Keep the server running until a keyboard interrupt
            time.sleep(1)
    except KeyboardInterrupt:
        logging.info("Stopping server...")
        logging.info(f"Disconnecting from {srv.active_clients_count} clients...")
        srv.stop()

        while srv.active_clients_count > 0:  # Wait for all clients to disconnect
            logging.info(f"Waiting for {srv.active_clients_count} clients to disconnect...")
            time.sleep(1)
        logging.info("Server stopped.")

        logging.info("THE END")

if __name__ == '__main__':
    main()
