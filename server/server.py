#! python3.7

import argparse
import io
import os
import speech_recognition as sr
import whisper
import torch
import logging

from datetime import datetime, timedelta
from queue import Queue
from tempfile import NamedTemporaryFile
from time import sleep
from sys import platform

from StreamServer import Server

logging.basicConfig(level=logging.DEBUG)


class Client:
    def __init__(self, client):

        self._client = client

        # The last time a recording was retreived from the queue.
        self.phrase_time = None
        # Current raw audio bytes.
        self.last_sample = bytes()
        # Thread safe Queue for passing data from the threaded recording callback.
        self.data_queue = Queue()

        self.transcription = ['']

        self.phrase_complete = False

        self.phrase_time = None

        self.temp_file = NamedTemporaryFile().name

    def send(self, data):
        self._client.send_message(data)



def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", default="medium", help="Model to use",
                        choices=["tiny", "base", "small", "medium", "large"])
    parser.add_argument("--non_english", action='store_true',
                        help="Don't use the english model.")
    parser.add_argument("--energy_threshold", default=1000,
                        help="Energy level for mic to detect.", type=int)
    parser.add_argument("--record_timeout", default=2,
                        help="How real time the recording is in seconds.", type=float)
    parser.add_argument("--phrase_timeout", default=3,
                        help="How much empty space between recordings before we "
                             "consider it a new line in the transcription.", type=float)
    parser.add_argument("--task", default="transcribe", help="transcribe or translate it to english",
                        choices=["transcribe", "translate"])


    if 'linux' in platform:
        parser.add_argument("--default_microphone", default='pulse',
                            help="Default microphone name for SpeechRecognition. "
                                 "Run this with 'list' to view available Microphones.", type=str)
    args = parser.parse_args()
    
    client_dict = {}

    client_queue = Queue()

    # # The last time a recording was retreived from the queue.
    # phrase_time = None
    # # Current raw audio bytes.
    # last_sample = bytes()
    # # Thread safe Queue for passing data from the threaded recording callback.
    # data_queue = Queue()
        
    # Load / Download model
    model = args.model
    if args.model != "large" and not args.non_english:
        model = model + ".en"
    audio_model = whisper.load_model(model)

    record_timeout = args.record_timeout
    phrase_timeout = args.phrase_timeout

    # temp_file = NamedTemporaryFile().name
    # transcription = ['']

    SECRET_TOKEN = "your_secret_token"
    srv = Server("127.0.0.1", 5000, 5001, SECRET_TOKEN, 4096, 5, 10, 1024)
    def OnConnected(c):
        print("Connected by", c.tcp_address())

        c.on_disconnected(lambda c: 
            print("Disconnected by", c.tcp_address())
        )

        c.on_timeout(lambda c:
            print("Timeout by", c.tcp_address())
        )

        newclient = Client(c)
        client_dict[c] = newclient

        def onmsg(c, data):
            if not c in client_dict:
                return
            client = client_dict[c]
            client.data_queue.put(data)
            
            if not client in client_queue.queue:
                client_queue.put(client)

        c.on_udp_message(onmsg)
    
    srv.on_connected(OnConnected)
    print("Starting server: 127.0.0.1:5000...")
    srv.start()




    # Cue the user that we're ready to go.
    print("Model loaded.\n")

    while True:
        try:
            # Infinite loops are bad for processors, must sleep.
            sleep(0.25)

            now = datetime.utcnow()
            if not client_queue.empty():
                client = client_queue.get()

                # Pull raw recorded audio from the queue.
                if not client.data_queue.empty():
                    client.phrase_complete = False
                    # If enough time has passed between recordings, consider the phrase complete.
                    # Clear the current working audio buffer to start over with the new data.
                    if client.phrase_time and now - client.phrase_time > timedelta(seconds=phrase_timeout):
                        client.last_sample = bytes()
                        client.phrase_complete = True
                    # This is the last time we received new audio data from the queue.
                    client.phrase_time = now

                    # Concatenate our current audio data with the latest audio data.
                    while not client.data_queue.empty():
                        data = client.data_queue.get()
                        client.last_sample += data

                    # Use AudioData to convert the raw data to wav data.
                    audio_data = sr.AudioData(client.last_sample, 48000, 2)
                    wav_data = io.BytesIO(audio_data.get_wav_data())

                    # Write wav data to the temporary file as bytes.
                    with open(client.temp_file, 'w+b') as f:
                        f.write(wav_data.read())

                    # Read the transcription.
                    result = audio_model.transcribe(client.temp_file, fp16=torch.cuda.is_available(), task = args.task)
                    text = result['text'].strip()

                    # If we detected a pause between recordings, add a new item to our transcripion.
                    # Otherwise edit the existing one.
                    if client.phrase_complete:
                        client.transcription.append(text)
                    else:
                        client.transcription[-1] = text

                    # send text to client
                    try:
                        tx = ""
                        for line in client.transcription:
                            tx = tx + line
                        client.send(str.encode(tx))
                    except:
                        pass

                    # # Clear the console to reprint the updated transcription.
                    # os.system('cls' if os.name=='nt' else 'clear')
                    # for line in transcription:
                    #     print(line)
                    # # Flush stdout.
                    # print('', end='', flush=True)

        # if KeyboardInterrupt stop. If everything else stop and show error.
        except (KeyboardInterrupt, Exception) as e:
            if isinstance(e, KeyboardInterrupt):
                logging.info("Stopping...")
            else:
                logging.error(e)
            break

    srv.stop()
    # print("\n\nTranscription:")
    # for line in transcription:
    #     print(line)


if __name__ == "__main__":
    main()