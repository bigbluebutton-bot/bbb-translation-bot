import os
from queue import Queue
import threading
from tempfile import NamedTemporaryFile

# Client class for each connected Client to handle the data seperatly.
class Client:
    def __init__(self, client):

        self.mutex = threading.Lock()

        self._client = client

        # The last time a recording was retreived from the queue.
        self.phrase_time = None
        # Current raw audio bytes.
        self.last_sample = bytes()
        # Thread safe Queue for passing data from the threaded recording callback.
        self.data_queue = Queue()

        self.oggs_opus_header_frames = bytes()
        self.oggs_opus_header_frames_complete = False

        self.transcription  = ""

        self.phrase_complete = False

        self.phrase_time = None

        self.temp_file = NamedTemporaryFile().name + ".opus"

    def send(self, data):
        self._client.send_message(data)

    def clear_buffer(self):
        if self.oggs_opus_header_frames_complete:
            with self.mutex:
                self.phrase_time = None
                os.remove(self.temp_file)
                self.temp_file = NamedTemporaryFile().name

                self.last_sample = self.oggs_opus_header_frames

    def stop(self):
        with self.mutex:
            self._client.stop()
            self.data_queue = Queue()
            self.last_sample = bytes()