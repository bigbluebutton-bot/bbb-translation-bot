import socket
import threading
import time

# Server
# The server will listen to incomming socket connections to receive the audio stream from the client
class Server:
    def __init__(self, host, port, timeout=5, backlog=5):
        self.host = host
        self.port = port
        self.backlog = backlog
        self.timeout = timeout

    def start(self):
        self.thread = threading.Thread(target=self.__start)
        self.thread.start()

    def __start(self):
        print("Starting server: " + self.host + ":" + str(self.port) + "...")

        self.running = True
        self.clients = []

        self.socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.socket.bind((self.host, self.port))
        self.socket.listen(self.backlog)

        print("Waiting for connections...")
        while self.running:
            try:
                conn, addr = self.socket.accept()
                if conn and self.running:
                    print("Connected by", addr)
                    client = self.__Client(self.__remove_client, conn, addr, self.timeout)
                    self.clients.append(client)
            except socket.timeout:
                pass

    def __remove_client(self, client):
        self.clients.remove(client)

    def stop(self):
        print("Stopping server...")
        self.running = False
        self.socket.close()

        # open a conection to close the socket
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.connect((self.host, self.port))
            s.close()
        except:
            pass

        # stop all clients
        clinetslist = self.clients.copy()
        for client in clinetslist:
            client.stop()


    # create a new thread to handle the connection
    class __Client:
        def __init__(self, method_to_remove_client, conn, addr, timeout=5):
            self.method_to_remove_client = method_to_remove_client
            self.conn = conn
            self.addr = addr
            self.timeout = timeout
            self.thread = threading.Thread(target=self.start)
            self.thread.start()

        running = True
        def start(self):
            self.last_recv = time.time()
            while self.running:
                if time.time() - self.last_recv > self.timeout:
                    print("Connection of " + str(self.addr) + " timed out.")
                    self.stop()
                    return
                data = self.conn.recv(1024)
                if data:
                    self.last_recv = time.time()
                    print(data)

        def stop(self):
            print("Stopping client " + str(self.addr))
            self.running = False
            self.conn.close()
            self.method_to_remove_client(self)


def main():
    SRV = Server('localhost', 5000, 5)
    SRV.start()
    time.sleep(10)
    SRV.stop()
    print("ende")

if __name__ == '__main__':
    main()