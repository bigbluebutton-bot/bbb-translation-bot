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

        self.OnConnectedCallback = []

    socket = None
    def start(self):
        if self.socket != None:
            return

        self.running = True
        self.clients = []
        try:
            self.socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            self.socket.bind((self.host, self.port))
            self.socket.listen(self.backlog)
        except:
            raise RuntimeError("Failed to start server")
            return

        self.thread = threading.Thread(target=self.__start)
        self.thread.start()

    def __start(self):
        while self.running:
            try:
                conn, addr = self.socket.accept()
                if conn and self.running:
                    client = self.__Client(self.__remove_client, conn, addr, self.timeout)
                    self.clients.append(client)

                    self.EmitConnected(client)
                    client.start()
            except socket.timeout:
                pass

    def __remove_client(self, client):
        self.clients.remove(client)

    def stop(self):
        if self.socket == None:
            return
        
        self.running = False
        if hasattr(self.socket, '_sock'):
            self.socket._sock.close()
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

        self.socket.close()
        self.socket = None

    # Events
    def OnConnected(self, callback):
        self.OnConnectedCallback.append(callback)

    def EmitConnected(self, client):
        if self.OnConnectedCallback:
            for callback in self.OnConnectedCallback:
                callback(client)


    # create a new thread to handle the connection
    class __Client:
        def __init__(self, method_to_remove_client, conn, addr, timeout=5):
            self.method_to_remove_client = method_to_remove_client
            self.conn = conn
            self.addr = addr
            self.timeout = timeout

            self.OnDisconnectedCallback = []
            self.OnTimeoutCallback = []
            self.OnMessageCallback = []

        def start(self):
            if self.conn == None or self.running == True:
                return
            self.thread = threading.Thread(target=self.__start)
            self.thread.start()

        running = False
        def __start(self):
            self.running = True
            self.last_recv = time.time()
            while self.running:
                if time.time() - self.last_recv > self.timeout:
                    self.EmitTimeout()
                    self.stop()
                    return
                try:
                    data = self.conn.recv(1024)
                except:
                    self.stop()
                    return
                if data:
                    self.last_recv = time.time()
                    self.EmitMessage(data)

        def stop(self):
            if self.conn == None or self.running == False:
                return
            self.EmitDisconnected()
            self.running = False
            try:
                self.conn.shutdown(socket.SHUT_RDWR)
                if hasattr(self.conn, '_sock'):
                    self.conn._sock.close()
                self.conn.close()
            except:
                pass
            self.conn = None
            self.method_to_remove_client(self)

        def Send(self, data):
            if self.conn:
                try:
                    self.conn.sendall(data)
                except:
                    pass

        # Events
        def OnDisconnected(self, callback):
            self.OnDisconnectedCallback.append(callback)
        
        def EmitDisconnected(self):
            if self.OnDisconnectedCallback:
                for callback in self.OnDisconnectedCallback:
                    thread = threading.Thread(target=callback)
                    thread.start()

        def OnTimeout(self, callback):
            self.OnTimeoutCallback.append(callback)
        
        def EmitTimeout(self):
            if self.OnTimeoutCallback:
                for callback in self.OnTimeoutCallback:
                    thread = threading.Thread(target=callback)
                    thread.start()

        def OnMessage(self, callback):
            self.OnMessageCallback.append(callback)
        
        def EmitMessage(self, data):
            if self.OnMessageCallback:
                for callback in self.OnMessageCallback:
                    thread = threading.Thread(target=callback, args=(data,))
                    thread.start()





# The OnConnected event will be called when a client connects to the server
# !!!IMPORTANT!!!
# The OnConnected event will not be called in a new thread
# If you block OnConnected, the server will not be able to accept new connections (deadlock)
# This is because the server is waiting for the OnConnected event to finish, so the client events can be created
# If you need to do something that takes a long time, create a new thread
# !!!IMPORTANT!!!
def OnConnected(client):
    print("Connected by", client.addr)

    client.OnDisconnected(lambda: 
        print("Disconnected by", client.addr)
    )

    client.OnTimeout(lambda:
        print("Timeout by", client.addr)
    )

    def onmessage(data):
        print("Message from", client.addr, ": ", data)
        client.Send(data)
    client.OnMessage(onmessage)



def main():
    SRV = Server('localhost', 5000, 5)
    SRV.OnConnected(OnConnected)

    print("Starting server: 127.0.0.1:5000...")
    SRV.start()
    print("Waiting for connections...")

    time.sleep(10)
    print("Stopping server...")
    SRV.stop()
    print("THE END")

if __name__ == '__main__':
    main()