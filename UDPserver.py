import socket
import logging
import time

BUFFER_SIZE = 1024
logging.basicConfig(level=logging.INFO)

SECRET_TOKEN = "your_secret_token"

class UDPServer:
    def __init__(self, host, port):
        logging.debug("Initializing UDPServer.")
        self.host = host
        self.port = port
        self.sessions = {}  # dictionary to keep track of client sessions
        self.socket = None
        self.running = False

    def start(self):
        self.running = True
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.socket.bind((self.host, self.port))

        while self.running:
            data, addr = self.socket.recvfrom(BUFFER_SIZE)
            logging.debug(f"Received {data} from {addr}")

            if addr not in self.sessions:
                if data.decode('utf-8') == SECRET_TOKEN:
                    logging.info(f"Valid token received from {addr}. Session authorized.")
                    self.sessions[addr] = True
                else:
                    logging.warning(f"Invalid token from {addr}. Ignoring.")
            else:
                self.handle_message(addr, data)

    def handle_message(self, addr, data):
        logging.info(f"Received from {addr}: {data.decode('utf-8')}")
        self.socket.sendto(b"OK", addr)

    def stop(self):
        self.running = False
        self.socket.close()

def main():
    srv = UDPServer('localhost', 5000)
    logging.info("Starting UDP server: 127.0.0.1:5000...")
    try:
        srv.start()
    except KeyboardInterrupt:
        logging.info("Stopping server...")
        srv.stop()
        logging.info("Server stopped.")
        logging.info("THE END")

if __name__ == '__main__':
    main()
