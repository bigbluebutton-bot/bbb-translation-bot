import socket
import time

import argparse
import io
import os
import speech_recognition as sr

from datetime import datetime, timedelta
from queue import Queue
from tempfile import NamedTemporaryFile
from time import sleep
from sys import platform


parser = argparse.ArgumentParser()
parser.add_argument("--record_timeout", default=2,
                    help="How real time the recording is in seconds.", type=float)
parser.add_argument("--default_microphone", default='pulse',
                    help="Default microphone name for SpeechRecognition. "
                            "Run this with 'list' to view available Microphones.", type=str)
args = parser.parse_args()

record_timeout = args.record_timeout


# Thread safe Queue for passing data from the threaded recording callback.
data_queue = Queue()
# We use SpeechRecognizer to record our audio because it has a nice feauture where it can detect when speech ends.
recorder = sr.Recognizer()
# recorder.energy_threshold = args.energy_threshold
# Definitely do this, dynamic energy compensation lowers the energy threshold dramtically to a point where the SpeechRecognizer never stops recording.
recorder.dynamic_energy_threshold = False

# Important for linux users. 
# Prevents permanent application hang and crash by using the wrong Microphone
print(platform)
if 'linux' in platform:
    mics = enumerate(sr.Microphone.list_microphone_names())
    print("Available microphone devices are: ")
    print("------------------------------------")
    for index, name in mics:
        print(f"Microphone with name \"{name}\" found")
    print("------------------------------------") 

    mic_name = args.default_microphone
    # mic_name = 'GENERAL WEBCAM: USB Audio (hw:2,0)'
    print(mic_name)
    if not mic_name or mic_name == 'list': 
        exit()
    else:
        for index, name in enumerate(sr.Microphone.list_microphone_names()):
            if mic_name in name:
                source = sr.Microphone(sample_rate=16000, device_index=index)
                break
else:
    source = sr.Microphone(sample_rate=16000)

print(source.SAMPLE_RATE, source.SAMPLE_WIDTH)

def record_callback(_, audio:sr.AudioData) -> None:
    """
    Threaded callback function to recieve audio data when recordings finish.
    audio: An AudioData containing the recorded bytes.
    """
    # Grab the raw bytes and push it into the thread safe queue.
    data = audio.get_raw_data()
    data_queue.put(data)

# Create a background thread that will pass us raw audio bytes.
# We could do this manually but SpeechRecognizer provides a nice helper.
recorder.listen_in_background(source, record_callback, phrase_time_limit=record_timeout)

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.connect(('localhost', 5000))


while True:
    try:
        # Pull raw recorded audio from the queue.
        if not data_queue.empty():
            while not data_queue.empty():
                data = data_queue.get()
                s.sendall(data)
    except KeyboardInterrupt:
        break

s.close()
