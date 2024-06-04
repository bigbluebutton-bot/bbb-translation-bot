#! python3.7
import io
import threading
import whisper
import torch
import logging
from pydub import AudioSegment
from datetime import datetime, timedelta
from queue import Queue
from time import sleep
from flask import Flask
import tempfile
from prometheus_client import Counter, Histogram, Gauge, start_http_server

from Config import load_settings
from StreamServer import Server
from extract_ogg import get_header_frames as extract_ogg_header_frames
from Client import Client




# Set logging level
logging.basicConfig(level=logging.INFO)




# Health check http sever
app = Flask(__name__)
STATUS = "stopped" # starting, running, stopping, stopped
@app.route('/health', methods=['GET'])
def healthcheck():
    global STATUS
    logging.info(STATUS)
    if STATUS == "running":
        return STATUS, 200
    else:
        return STATUS, 503




# Metrics definitions for prometheus
# http://127.0.0.1:1000/graph?g0.expr=connected_clients&g0.tab=0&g0.display_mode=stacked&g0.show_exemplars=0&g0.range_input=15m&g1.expr=avg(rate(speech_processing_time_seconds_sum%5B1m%5D)%20%2F%20rate(speech_processing_time_seconds_count%5B1m%5D))&g1.tab=0&g1.display_mode=stacked&g1.show_exemplars=0&g1.range_input=15m&g2.expr=avg(rate(client_queue_wait_time_seconds_sum%5B1m%5D)%20%2F%20rate(client_queue_wait_time_seconds_count%5B1m%5D))&g2.tab=0&g2.display_mode=stacked&g2.show_exemplars=0&g2.range_input=15m&g3.expr=avg(rate(total_processing_time_seconds_sum%5B1m%5D)%20%2F%20rate(total_processing_time_seconds_count%5B1m%5D))&g3.tab=0&g3.display_mode=stacked&g3.show_exemplars=0&g3.range_input=15m
connected_clients = Gauge('connected_clients', 'Number of currently connected clients')
whisper_workers = Gauge('whisper_workers', 'Number of whisper workers transcribing/translating speech')
processing_time = Histogram('speech_processing_time_seconds', 'Time taken to process speech')
queue_wait_time = Histogram('client_queue_wait_time_seconds', 'Time a client has waited in the queue')
total_processing_time = Histogram('total_processing_time_seconds', 'Time from receiving audio to sending transcription')

# Vars
client_dict = {}        # Dictionary with all connected clients
client_queue = Queue()  # Queue with clients which have data to process
client_dict_mutex = threading.Lock() # Mutex to lock the client_dict
client_queue_mutex = threading.Lock() # Mutex to lock the client_queue

# Load settings from env variables use default values if not set
settings = load_settings()

loading_model_mutex = threading.Lock()
class worker:
    def __init__(self, model_type = "tiny", english_only = "false", model_path = ""):
        if english_only:
            model_type = model_type + ".en"
        self.model_type = model_type
        self.model = None
        self.running_mutex = threading.Lock()
        self.model_path = model_path
        self.running = False

    def process(self):
        if self.running_mutex.locked():
            return # return if mutex is locked
        
        # Load model
        with self.running_mutex:
            if self.model_path != "":
                with loading_model_mutex:
                    logging.info(f"Loading model '{self.model_type}'...")
                    self.model = whisper.load_model(self.model_type, download_root=self.model_path)
                    logging.info("Model loaded")
            else:
                self.model = whisper.load_model(self.model_type)

            # Start processing
            self.running = True

            whisper_workers.inc()  # Increment whisper workers for prometheus

            time_to_sleep = 0.25
            while self.running:
                client = None
                try:
                    sleep(time_to_sleep)    # Infinite loops are bad for processors, must sleep.

                    # If there is no data to process, sleep for a bit.
                    if client_queue.empty():
                        time_to_sleep = 0.25    # Set sleep time to 0.25 seconds if there is no data to process.
                    else:
                        time_to_sleep = 0.0     # Set sleep time to 0 seconds if there is data to process.

                        # Process data from client
                        processing_start_time = datetime.utcnow()
                        client = None
                        with client_queue_mutex:
                            client = client_queue.get()
                            queue_wait_time.observe((datetime.utcnow() - client.client_queue_wait_time).total_seconds()) # Observe the time the client has waited in the queue for prometheus
                            time_data_received = client.time_data_received


                        last_sample = None

                        # Pull raw recorded audio from the queue.
                        with client.mutex:
                            if not client.data_queue.empty():
                                if client.phrase_time is None:
                                    client.phrase_time = processing_start_time

                                # Concatenate our current audio data with the latest audio data.
                                while not client.data_queue.empty():
                                    data = client.data_queue.get()
                                    client.last_sample += data

                            last_sample = client.last_sample

                            # set header
                            if client.oggs_opus_header_frames_complete == False:
                                id_header_frame, comment_header_frames = extract_ogg_header_frames(last_sample)
                                if id_header_frame is not None and len(comment_header_frames) > 0:
                                    client.oggs_opus_header_frames += id_header_frame.raw_data
                                    for frame in comment_header_frames:
                                        client.oggs_opus_header_frames += frame.raw_data
                                    client.oggs_opus_header_frames_complete = True
                                continue

                        # # Write to file for debugging.
                        # with open('/testing/sample.opus', 'wb') as f:
                        #     f.write(last_sample)

                        # Create temporary file for audio data which will be deleted after use.
                        with tempfile.NamedTemporaryFile(prefix='tmp_audio_', suffix='.wav', dir=settings['RAM_DISK_PATH'], delete=True) as temp_file:
                            # Convert opus to wav
                            opus_data = io.BytesIO(last_sample)
                            opus_audio = AudioSegment.from_file(opus_data, format="ogg", frame_rate=48000, channels=2, sample_width=2)
                            opus_audio.export(temp_file.name, format="wav")
                            
                            # Transcribe audio data
                            result = self.model.transcribe(temp_file.name, fp16=torch.cuda.is_available(), task=settings["TASK"])
                            text = result['text'].strip()
                            # logging.info(str.encode(text))

                        # Set transcription
                        with client.mutex:
                            client.transcription = text

                            # Send text to client
                            try:
                                client.send(str.encode(text))
                            except:
                                pass


                        # If enough time has passed between recordings, consider the phrase complete.
                        # Clear the current working audio buffer to start over with the new data.
                        if client.phrase_time and processing_start_time - client.phrase_time > timedelta(seconds=settings["RECORD_TIMEOUT"]):
                            # logging.info("Clear audio buffer")
                            client.clear_buffer()

                        processing_time.observe((datetime.utcnow() - processing_start_time).total_seconds())  # Observe processing time for prometheus
                        total_processing_time.observe((datetime.utcnow() - time_data_received).total_seconds())  # Observe total processing time for prometheus
                except Exception as e:
                    logging.error(e)
                    if client:
                        client.stop()
                        break
        
        whisper_workers.dec()  # Decrement whisper workers for prometheus
        self.stop()

    def stop(self):
        if self.running_mutex.locked():
            self.running = False
            with self.running_mutex:
                self.model = None


    




# Main function which handles all clients one by one (Queue).
def main():
    global STATUS
    STATUS = "starting"

    # Start the health http-server (flask) in a new thread.
    webserverthread = threading.Thread(target=app.run, kwargs={'debug': False, 'host': settings["HOST"], 'port': settings["HEALTH_CHECK_PORT"]})
    webserverthread.daemon = True  # This will ensure the thread stops when the main thread exits
    webserverthread.start()
    

    # Start the prometheus http-server
    start_http_server(settings["PROMETHEUS_PORT"])

        
    # Load / Download model
    model = settings["MODEL"]
    if settings["MODEL"] != "large" and settings["ONLY_ENGLISH"]:
        model = model + ".en"

    # Create server
    srv = Server(settings["HOST"], settings["TCPPORT"], settings["UDPPORT"], settings["SECRET_TOKEN"], 4096, 5, 10, 1024, settings["EXTERNALHOST"])

    # Handle new connections and disconnections, timeouts and messages
    def OnConnected(c):
        logging.info(f"Connected by {c.tcp_address()}")
        connected_clients.inc()  # Increment on new connection

        # Create new client
        newclient = Client(c)
        with client_dict_mutex:
            client_dict[c] = newclient

        # Handle disconnections
        def ondisconnedted(c):
            logging.info(f"Disconnected by {c.tcp_address()}")
            connected_clients.dec()  # Decrement on disconnection
            # Remove client from client_dict
            with client_dict_mutex:
                if c in client_dict:
                    del client_dict[c]
        c.on_disconnected(ondisconnedted)

        # Handle timeouts
        def ontimeout(c):
            logging.info(f"Timeout by {c.tcp_address()}")
            # Remove client from client_dict
            with client_dict_mutex:
                if c in client_dict:
                    del client_dict[c]
        c.on_timeout(ontimeout)

        # Handle messages
        def onmsg(c, data):
            logging.debug(f"UDP from: {c.tcp_address()}")
            time_data_received = datetime.utcnow()
            with client_dict_mutex:
                if not c in client_dict:
                    logging.error(f"Client {c.tcp_address()} not in list!")
                    return
                client = client_dict[c]
                with client.mutex:
                    logging.debug(f"Add data to data queue for client {c.tcp_address()}")
                    client.data_queue.put(data)
                
                # Add client to client_queue if not already in it
                with client_queue_mutex:
                    if not client in client_queue.queue:
                        logging.debug(f"Add client to client queue for client {c.tcp_address()}")
                        client.client_queue_wait_time = datetime.utcnow()
                        client.time_data_received = time_data_received
                        client_queue.put(client)
        c.on_udp_message(onmsg)
    srv.on_connected(OnConnected)

    # Start server
    logging.info(f"Starting server: {settings['HOST']}:{settings['TCPPORT']}...")
    srv.start()
    logging.info("Ready to transcribe. Press Ctrl+C to stop.")


    # create worker threaded
    workers = []
    for _ in range(2):
        w = worker(model_type=settings["MODEL"], english_only=settings["ONLY_ENGLISH"], model_path=settings["MODEL_PATH"])
        t = threading.Thread(target=w.process)
        t.daemon = True
        t.start()
        workers.append(w)

    STATUS = "running"

    # Wait until stopped by Strg + C
    try:
        while True:
            sleep(0.25)
    except KeyboardInterrupt:
        pass


    # Stop server
    STATUS = "stopping"
    logging.info("Stopping server...")
    for w in workers:
        w.stop()
    srv.stop()
    logging.info("Server stopped")
    STATUS = "stopped"


if __name__ == "__main__":
    main()