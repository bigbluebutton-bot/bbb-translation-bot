import os
import socket
import threading
import logging
import inspect
import time

from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend

BUFFER_SIZE = 1024

logging.basicConfig(level=logging.DEBUG)




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
    def __init__(self, host, port, encryption=False, aes_key=5, aes_initkey=5):
        logging.debug("Initializing Server.")
        self.host = host
        self.port = port
        self._running = False
        self._socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.encryption = encryption
        self.aes_key = aes_key  # 32 bytes AES key
        self.aes_initkey = aes_initkey
        self.whitelist = []
        self._message_callbacks = EventHandler()
        self._main_thread = None
        
    def _handle_socket_errors(self, error):
        """Centralize error handling for socket-related errors."""
        logging.debug(f"Socket error: {error}")
        self.stop()


    def start(self):
        self._socket.bind((self.host, self.port))
        logging.info(f"Server started at {self.host}:{self.port}")
        self._running = True
        t = threading.Thread(target=self._listen)
        self._main_thread = t
        t.start()



    def _listen(self):
        while self._running:
            try:
                data, addr = self._socket.recvfrom(BUFFER_SIZE)
                if not len(self.whitelist) == 0:
                    if not addr[0] in self.whitelist:
                        continue
                if data:
                    if self.encryption:
                        data = self._decrypt(data)

                    self._message_callbacks.emit(addr, data)

            except (Exception) as e:
                if not self._running:
                    return
                self._handle_socket_errors(e)

    def stop(self):
        """Stop the server."""
        if not self._running:
            logging.warning("Server already stopped.")
            return
        self._running = False
        self._socket.close()
        self._main_thread.join()
        logging.info("Server stopped.")


    def _decrypt(self, encrypted_data):
        """Decrypt the received data."""
        logging.debug(f"Decrypting data")

        cipher = Cipher(algorithms.AES(self.aes_key), modes.CFB(self.aes_initkey), backend=default_backend())

        decryptor = cipher.decryptor()

        plaintext = decryptor.update(encrypted_data) + decryptor.finalize()
        return plaintext


    def on_event(self, event_type, callback):
        # Get the number of parameters the callback has
        num_params = len(inspect.signature(callback).parameters)

        if event_type == 'message':
            if num_params == 2:
                return self._message_callbacks.add_event(callback)
            else:
                logging.error(f"Invalid number of parameters for 'message' event. Expected 2, got {num_params}.")

    def remove_event(self, event_type, event_id):
        """Remove an event callback on the event type."""
        if event_type == "message":
            self._message_callbacks.remove_event(event_id)
        else:
            logging.warning(f"Unsupported event type: {event_type}")



# Usage:
def main():
    aes_key = os.urandom(32)  # Generate a random 32 bytes AES key
    aes_initkey = os.urandom(16) # Generate a random 16 bytes AES init key
    print(f"AES Key: {aes_key}")
    print(f"AES Init Key: {aes_initkey}")
    srv =  Server('localhost', 5001, True, aes_key, aes_initkey)

    srv.whitelist = ['127.0.0.1']

    srv.on_event("message", lambda addr, data: logging.info(f"msg from {addr}: {data}"))

    srv.start()

    try:
        while True:  # Keep the server running until a keyboard interrupt
            time.sleep(1)
    except KeyboardInterrupt:
        logging.info("Stopping server...")
        srv.stop()

if __name__ == '__main__':
    main()