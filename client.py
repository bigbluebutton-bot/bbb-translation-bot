import socket
import time


# Connect to localhost:5000 and send a message
def connect():
    for i in range(10):
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.connect(('localhost', 5000))
            try:
                s.sendall(b'Hello, world' + str(i).encode())
            except:
                print("Send failed")

            try:
                s.close()
            except:
                print("Close failed")
        except:
            print("Connection failed")

connect()
