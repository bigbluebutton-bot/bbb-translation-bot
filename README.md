# BBB-Translation-Bot

BBB-Translation-Bot is a tool designed to enhance communication in BigBlueButton (BBB) meetings by providing real-time transcription and translation services. Utilizing OpenAI's cutting-edge [Whisper](https://github.com/openai/whisper) technology, the bot joins a BBB meeting's audio channel and transcribes/translates spoken words into text, seamlessly integrating the transcripts into BBB's closed captions feature.

# Getting Started

Follow these simple steps to quickly set up the BBB-Translation-Bot.

## Prerequisites

### Hardware
This setup was testet with a Nvidia RTX 2070 and RTX 3070 GPU. The GPU is used for the transcription and translation of the audio stream.

### Install Nvidia drivers for Ubuntu:
Refer to the official [Nvidia documentation](https://docs.nvidia.com/datacenter/tesla/tesla-installation-notes/index.html#ubuntu-lts):
```bash
sudo apt update
sudo apt install linux-headers-$(uname -r)
distribution=$(. /etc/os-release;echo $ID$VERSION_ID | sed -e 's/\.//g')
wget https://developer.download.nvidia.com/compute/cuda/repos/$distribution/x86_64/cuda-keyring_1.0-1_all.deb
sudo dpkg -i cuda-keyring_1.0-1_all.deb
sudo apt update
sudo apt -y install cuda-drivers
sudo reboot now
```

### Docker with GPU support
Ensure Docker with GPU support is installed on your system:
Refer to the official [Nvidia documentation](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html)

```bash
sudo apt update

curl -sSL https://get.docker.com | sh

curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
  && curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
    sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
    sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list \
  && \
    sudo apt-get update

sudo apt install nvidia-container-runtime

which nvidia-container-runtime-hook

sudo systemctl restart docker

docker run -it --rm --gpus all ubuntu nvidia-smi # Test GPU support
```

### Installation
1. Clone the repository:
    ```bash
    git clone https://github.com/bigbluebutton-bot/bbb-translation-bot
    cd bbb-translation-bot
    ```

2. Configure the bot:
Copy the example configuration file and modify it according to your preferences:
    ```bash
    cp .env_example .env
    ```

3. Launch the bot:
With Docker, you can easily start the bot:
    ```bash	
    docker-compose up -d
    ```

# Development Setup
To set up the environment for development purposes, follow the instructions below.
## Server Setup
1. Install Nvidia drivers for Ubuntu:
Refer to the official [Nvidia documentation](https://docs.nvidia.com/datacenter/tesla/tesla-installation-notes/index.html#ubuntu-lts):
    ```bash
    sudo apt install linux-headers-$(uname -r)
    distribution=$(. /etc/os-release;echo $ID$VERSION_ID | sed -e 's/\.//g')
    wget https://developer.download.nvidia.com/compute/cuda/repos/$distribution/x86_64/cuda-keyring_1.0-1_all.deb
    sudo dpkg -i cuda-keyring_1.0-1_all.deb
    sudo apt update
    sudo apt -y install cuda-drivers
    sudo reboot now
    ```

2. Install Python dependencies:
    ```bash
    cd server
    sudo apt update
    sudo apt install python3-pip python3-dev ffmpeg -y
    python3 -m venv .translation-server
    source .translation-server/bin/activate
    pip3 install -r requirements-server.txt --no-cache-dir
    ```

3. Run the server:
    ```bash
    python3 server.py
    ```

4. Exit the virtual environment:
    ```bash
    deactivate
    ```

## Client Setup
1. Install Golang:
    Follow the official [Golang installation guide](https://go.dev/doc/install):
    ```bash
    cd client
    sudo apt update
    wget https://go.dev/dl/go1.21.3.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.3.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    ```

2. Install node.js:
Use [Node Version Manager](https://github.com/nvm-sh/nvm) to install Node.js:
   ```bash
   curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.5/install.sh | bash
   export NVM_DIR="$([ -z "${XDG_CONFIG_HOME-}" ] && printf %s "${HOME}/.nvm" || printf %s "${XDG_CONFIG_HOME}/nvm")"
   [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" # This loads nvm
   nvm install node
   ```

3. Install Go dependencies:
    ```bash
    go get .
    ```

4. Run the client:
    ```bash
    go run .
    ```
