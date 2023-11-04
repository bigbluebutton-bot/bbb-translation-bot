FROM pytorch/pytorch:2.1.0-cuda12.1-cudnn8-runtime AS build-env

RUN apt update && apt install -y git

WORKDIR /app
COPY *.py *.txt ./

RUN pip install -r requirements.txt

CMD ["python3", "server.py"]