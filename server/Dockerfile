FROM nvidia/cuda:12.2.2-devel-ubuntu22.04 AS build-env

RUN apt update && apt install -y git python3 python3-pip ffmpeg

WORKDIR /app
COPY *.txt ./


RUN pip install -r requirements.txt

COPY *.py ./

CMD ["python3", "server.py"]