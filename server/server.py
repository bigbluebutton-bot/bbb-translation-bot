#! python3.7
import io
import os
import ipaddress
import sys
import threading
import speech_recognition as sr
import whisper
import torch
import logging
from dotenv import load_dotenv
from pydub import AudioSegment
from datetime import datetime, timedelta
from queue import Queue
from tempfile import NamedTemporaryFile
from time import sleep
from sys import platform
from flask import Flask

from StreamServer import Server
from extract_ogg import get_header_frames as extract_ogg_header_frames

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)




def load_settings():
    # Load environment variables from .env file, if available
    load_dotenv()

    # if the config is valid
    valid_config = True

    def validate_model(value, default, env_var):
        nonlocal valid_config
        valid_models = ['tiny', 'base', 'small', 'medium', 'large']
        if value not in valid_models:
            logging.error(f"Invalid MODEL setting: {env_var}. Must be one of {valid_models}.")
            valid_config = False
            return default
        return value

    def validate_float(value, default, env_var):
        nonlocal valid_config
        try:
            return float(value)
        except ValueError:
            logging.error(f"Invalid float value for setting: {env_var}. Using default value: {default}")
            valid_config = False
            return default

    def validate_task(value, default, env_var):
        nonlocal valid_config
        valid_tasks = ['transcribe', 'translate']
        if value not in valid_tasks:
            logging.error(f"Invalid TASK setting: {env_var}. Must be one of {valid_tasks}.")
            valid_config = False
            return default
        return value

    def validate_int(value, default, env_var):
        nonlocal valid_config
        try:
            return int(value)
        except ValueError:
            logging.error(f"Invalid integer value for setting: {env_var}. Using default value: {default}")
            valid_config = False
            return default

    def validate_bool(value, default, env_var):
        nonlocal valid_config
        if value.lower() in ["true", "false"]:
            return value.lower() == "true"
        else:
            logging.error(f"Invalid boolean value for setting: {env_var}. Expected 'true' or 'false'. Using default value: {default}")
            valid_config = False
            return default

    def get_variable(env_var, default, validate_func=None):
        value = os.getenv(env_var, default)
        if validate_func:
            value = validate_func(value, default, env_var)
        return value

    settings = {
        'MODEL': get_variable('TRANSCRIPTION_SERVER_MODEL', "medium", validate_model),
        'ONLY_ENGLISH': get_variable('TRANSCRIPTION_SERVER_ONLY_ENGLISH', "false", validate_bool),
        'RECORD_TIMEOUT': get_variable('TRANSCRIPTION_SERVER_RECORD_TIMEOUT', 10.0, validate_float),
        'TASK': get_variable('TRANSCRIPTION_SERVER_TASK', "transcribe", validate_task),
        'HOST': get_variable('TRANSCRIPTION_SERVER_HOST', "0.0.0.0"),
        'EXTERNALHOST': get_variable('TRANSCRIPTION_SERVER_EXTERNAL_HOST', "127.0.0.1"),
        'TCPPORT': get_variable('TRANSCRIPTION_SERVER_PORT_TCP', 5000, validate_int),
        'UDPPORT': get_variable('TRANSCRIPTION_SERVER_PORT_UDP', 5001, validate_int),
        'SECRET_TOKEN': get_variable('TRANSCRIPTION_SERVER_SECRET', "your_secret_token")
    }

    if not valid_config:
        logging.error("Invalid config. Please fix the errors and try again.")
        sys.exit(1)

    return settings





# Health check http sever
STATUS = "stopped" # starting, running, stopping, stopped
@app.route('/health', methods=['GET'])
def healthcheck():
    global STATUS
    logging.info(STATUS)
    if STATUS == "running":
        return STATUS, 200
    else:
        return STATUS, 503





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




# Main function which handles all clients one by one (Queue).
def main():
    global STATUS
    STATUS = "starting"


    # Load settings from env variables or config file or use default values if not set
    settings = load_settings()
    # Print the settings for demonstration purposes
    for key, value in settings.items():
        logging.debug(f"{key}: {value}")

    # Start the health http-server (flask) in a new thread.
    webserverthread = threading.Thread(target=app.run, kwargs={'debug': False, 'host': settings["HOST"], 'port': 8001})
    webserverthread.daemon = True  # This will ensure the thread stops when the main thread exits
    webserverthread.start()
    
        
    # Load / Download model
    model = settings["MODEL"]
    if settings["MODEL"] != "large" and settings["ONLY_ENGLISH"]:
        model = model + ".en"
    logging.info(f"Loading model '{model}'...")
    audio_model = whisper.load_model(model, download_root="/app/models")
    logging.info("Model loaded")


    # Vars
    client_dict = {}        # Dictionary with all connected clients
    client_queue = Queue()  # Queue with clients which have data to process
    client_dict_mutex = threading.Lock() # Mutex to lock the client_dict
    client_queue_mutex = threading.Lock() # Mutex to lock the client_queue

    # Create server
    srv = Server(settings["HOST"], settings["TCPPORT"], settings["UDPPORT"], settings["SECRET_TOKEN"], 4096, 5, 10, 1024, settings["EXTERNALHOST"])

    # Handle new connections and disconnections, timeouts and messages
    def OnConnected(c):
        logging.info(f"Connected by {c.tcp_address()}")

        # Create new client
        newclient = Client(c)
        logging.info(f"TEMP: {newclient.temp_file}")
        with client_dict_mutex:
            client_dict[c] = newclient

        # Handle disconnections
        def ondisconnedted(c):
            logging.info(f"Disconnected by {c.tcp_address()}")
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
                        client_queue.put(client)
        c.on_udp_message(onmsg)
    srv.on_connected(OnConnected)

    # Start server
    logging.info(f"Starting server: {settings['HOST']}:{settings['TCPPORT']}...")
    srv.start()
    STATUS = "running"
    logging.info("Ready to transcribe. Press Ctrl+C to stop.")


    # Main loop to process incomming data from all clients
    time_to_sleep = 0.25    # Infinite loops are bad for processors, must sleep.
    while True:
        client = None
        try:
            sleep(time_to_sleep)    # Infinite loops are bad for processors, must sleep.

            # If there is no data to process, sleep for a bit.
            if client_queue.empty():
                time_to_sleep = 0.25    # Set sleep time to 0.25 seconds if there is no data to process.
            else:
                time_to_sleep = 0.0     # Set sleep time to 0 seconds if there is data to process.

                # Process data from client
                now = datetime.utcnow()
                client = None
                with client_queue_mutex:
                    client = client_queue.get()

                last_sample = None
                temp_file = None

                # Pull raw recorded audio from the queue.
                with client.mutex:
                    if not client.data_queue.empty():
                        if client.phrase_time is None:
                            client.phrase_time = now

                        # Concatenate our current audio data with the latest audio data.
                        while not client.data_queue.empty():
                            data = client.data_queue.get()
                            client.last_sample += data

                    last_sample = client.last_sample
                    temp_file = client.temp_file

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

                # Convert opus to wav
                opus_data = io.BytesIO(last_sample)
                opus_audio = AudioSegment.from_file(opus_data, format="ogg", frame_rate=48000, channels=2, sample_width=2)
                opus_audio.export(temp_file, format="wav")

                # Convert audio to text using the model (if translation is enabled translate to english)
                result = audio_model.transcribe(temp_file, fp16=torch.cuda.is_available(), task = settings["TASK"])
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
                if client.phrase_time and now - client.phrase_time > timedelta(seconds=settings["RECORD_TIMEOUT"]):
                    # logging.info("Clear audio buffer")
                    client.clear_buffer()

        # if KeyboardInterrupt stop. If everything else stop and show error.
        except (KeyboardInterrupt, Exception) as e:
            if isinstance(e, KeyboardInterrupt):
                break   # Stop if Ctrl+C is pressed
            else:
                logging.error(e)
                client.stop()


    # Stop server
    STATUS = "stopping"
    logging.info("Stopping server...")
    srv.stop()
    logging.info("Server stopped")
    STATUS = "stopped"


if __name__ == "__main__":
    main()