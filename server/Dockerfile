FROM pytorch/pytorch AS build-env

RUN apt update && apt install -y git

WORKDIR /app
COPY *.py *.txt ./

RUN pip install -r requirements.txt

CMD ["python3", "server.py"]