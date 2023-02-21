# Whisper Real-Time API

This is a simple API to transcript an audio stream to text using [Whisper](https://github.com/openai/whisper).

## Installation
### Server
```bash
sudo apt update

# Install python3 and pip3
sudo apt install python3-pip python3-dev -y

# Install ffmpeg
sudo apt install ffmpeg

# Install python dependencies
pip3 install -r requirements-server.txt --no-cache-dir
```

### Client
#### Python
```bash
sudo apt update

# Install python3 and pip3
sudo apt install python3-pip python3-dev -y

# Install portaudio and pyaudio
sudo apt install portaudio19-dev python3-pyaudio

# Install python dependencies
pip3 install -r requirements-client.txt --no-cache-dir
```

#### Golang
```bash
sudo apt update

# Install golang (https://go.dev/doc/install)
wget https://go.dev/dl/go1.20.1.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.20.1.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install go dependencies
go get ./...
```

## Usage
### Server
```bash
python3 server.py
```

### Client
#### Python
```bash
python3 client.py
```

#### Golang
```bash
go run client.go
```


## Source
- [davabase/whisper_real_time](https://github.com/davabase/whisper_real_time)

