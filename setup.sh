#!/bin/bash

SCREEN_NAME="install_session"
LOGIN_HINT_FILE="/etc/profile.d/install_hint.sh"

# Main execution: Define tasks and run the main function
main() {
    os_name=$(get_os)
    if [ $? -ne 0 ]; then
        echo "❌ Unsupported OS or version: $os_name"
        exit 1
    fi

    # Simple setup
    if $INSTALL_SIMPLE_SETUP; then
        if [ "$os_name" == "Ubuntu 22" ]; then
            # Ubuntu 22
            add_task "Install NVIDIA Drivers" nvidia_install check_nvidia_driver  # Skip if drivers are installed
            add_task "Install NVIDIA CUDA" cuda_install check_cuda_installed  # Skip if CUDA is installed
            add_task "Install NVIDIA CUDA Toolkit" cuda_toolkit check_toolkit  # Skip if toolkit is installed
            add_task "Reboot" configure_reboot_simple check_reboot_configured_simple  # Skip if reboot is needed
            add_task "Install Docker" install_docker check_docker_installed  # Skip if Docker is installed
            add_task "Install Docker with NVIDIA support" install_docker_nvidia check_docker_nvidia  # Skip if Docker with NVIDIA support is installed
        elif [ "$os_name" == "Ubuntu 22 WSL" ]; then
            # Ubuntu 22 WSL
            # Check the README for WSL
            exit 1
        elif [ "$os_name" == "Debian 12" ]; then
            # Debian 12
            add_task "Install NVIDIA Drivers" nvidia_install_debian check_nvidia_driver_debian  # Skip if drivers are installed
            add_task "Install NVIDIA CUDA" cuda_install_debian check_cuda_installed_debian  # Skip if CUDA is installed
            add_task "Install NVIDIA CUDA Toolkit" cuda_toolkit check_toolkit  # Skip if toolkit is installed
            add_task "Reboot" configure_reboot_simple check_reboot_configured_simple  # Skip if reboot is needed
            add_task "Install Docker" install_docker check_docker_installed  # Skip if Docker is installed
            add_task "Install Docker with NVIDIA support" install_docker_nvidia check_docker_nvidia  # Skip if Docker with NVIDIA support is installed
        elif [ "$os_name" == "Debian 12 WSL" ]; then
            # Debian 12 WSL
            # Check the README for WSL
            exit 1
        fi
    else
        # Full setup
        if [ "$os_name" == "Ubuntu 22" ]; then
            # Ubuntu 22
            add_task "Install NVIDIA Drivers" nvidia_install check_nvidia_driver  # Skip if drivers are installed
            add_task "Install NVIDIA CUDA" cuda_install check_cuda_installed  # Skip if CUDA is installed
            add_task "Install NVIDIA CUDA Toolkit" cuda_toolkit check_toolkit  # Skip if toolkit is installed
            add_task "Install NVIDIA cuDNN 8.9.7" install_cudnn check_cudnn_installed  # Skip if cuDNN is installed
            add_task "Reboot" configure_reboot_ubuntu22 check_reboot_configured_ubuntu22  # Skip if reboot is needed
            add_task "Install Docker" install_docker check_docker_installed  # Skip if Docker is installed
            add_task "Install Docker with NVIDIA support" install_docker_nvidia check_docker_nvidia  # Skip if Docker with NVIDIA support is installed
            add_task "Install ffmpeg" install_ffmpeg check_ffmpeg_installed  # Skip if ffmpeg is installed
            add_task "Install golang" install_golang check_golang_installed  # Skip if golang is installed
            add_task "Install python3" install_python3 check_python3_installed  # Skip if python3 is installed
            add_task "Install nodejs" install_nodejs check_nodejs_installed  # Skip if nodejs is installed
        elif [ "$os_name" == "Ubuntu 22 WSL" ]; then
            # Ubuntu 22 WSL
            add_task "Install NVIDIA CUDA Toolkit" cuda_toolkit check_toolkit  # Skip if toolkit is installed
            add_task "Install NVIDIA cuDNN 8.9.7" install_cudnn check_cudnn_installed  # Skip if cuDNN is installed
            add_task "Install ffmpeg" install_ffmpeg check_ffmpeg_installed  # Skip if ffmpeg is installed
            add_task "Install golang" install_golang check_golang_installed  # Skip if golang is installed
            add_task "Install python3" install_python3 check_python3_installed  # Skip if python3 is installed
            add_task "Install nodejs" install_nodejs check_nodejs_installed  # Skip if nodejs is installed
        elif [ "$os_name" == "Debian 12" ]; then
            # Debian 12
            add_task "Install NVIDIA Drivers" nvidia_install_debian check_nvidia_driver_debian  # Skip if drivers are installed
            add_task "Install NVIDIA CUDA" cuda_install_debian check_cuda_installed_debian  # Skip if CUDA is installed
            add_task "Install NVIDIA CUDA Toolkit" cuda_toolkit check_toolkit  # Skip if toolkit is installed
            add_task "Install NVIDIA cuDNN 8.9.7" install_cudnn check_cudnn_installed  # Skip if cuDNN is installed
            add_task "Reboot" configure_reboot_ubuntu22 check_reboot_configured_ubuntu22  # Skip if reboot is needed
            add_task "Install Docker" install_docker check_docker_installed  # Skip if Docker is installed
            add_task "Install Docker with NVIDIA support" install_docker_nvidia check_docker_nvidia  # Skip if Docker with NVIDIA support is installed
            add_task "Install ffmpeg" install_ffmpeg check_ffmpeg_installed  # Skip if ffmpeg is installed
            add_task "Install golang" install_golang check_golang_installed  # Skip if golang is installed
            add_task "Install python3" install_python3 check_python3_installed  # Skip if python3 is installed
            add_task "Install nodejs" install_nodejs check_nodejs_installed  # Skip if nodejs is installed
        elif [ "$os_name" == "Debian 12 WSL" ]; then
            # Debian 12 WSL
            add_task "Install NVIDIA CUDA Toolkit" cuda_toolkit check_toolkit  # Skip if toolkit is installed
            add_task "Install NVIDIA cuDNN 8.9.7" install_cudnn check_cudnn_installed  # Skip if cuDNN is installed
            add_task "Install ffmpeg" install_ffmpeg check_ffmpeg_installed  # Skip if ffmpeg is installed
            add_task "Install golang" install_golang check_golang_installed  # Skip if golang is installed
            add_task "Install python3" install_python3 check_python3_installed  # Skip if python3 is installed
            add_task "Install nodejs" install_nodejs check_nodejs_installed  # Skip if nodejs is installed
        fi
    fi

    # Process task-specific logic
    process_tasks
}

CHECK_DEPENDENCIES=false
check_dependencies() {
    # Simple setup
    if $INSTALL_SIMPLE_SETUP; then
        if [ "$os_name" == "Ubuntu 22" ]; then
            # Ubuntu 22
            check_nvidia_driver || exit 1
            check_cuda_installed || exit 1
            check_toolkit || exit 1
            check_docker_installed || exit 1
            check_docker_nvidia || exit 1
        elif [ "$os_name" == "Ubuntu 22 WSL" ]; then
            # Ubuntu 22 WSL
            check_nvidia_driver || exit 1
            check_docker_installed || exit 1
            check_docker_nvidia || exit 1
            exit 1
        elif [ "$os_name" == "Debian 12" ]; then
            # Debian 12
            check_nvidia_driver_debian || exit 1
            check_toolkit || exit 1
            check_docker_installed || exit 1
            check_docker_nvidia || exit 1
        elif [ "$os_name" == "Debian 12 WSL" ]; then
            # Debian 12 WSL
            check_toolkit || exit 1
            check_docker_installed || exit 1
            check_docker_nvidia || exit 1
            exit 1
        fi
    else
        # Full setup
        if [ "$os_name" == "Ubuntu 22" ]; then
            # Ubuntu 22
            check_nvidia_driver || exit 1
            check_cuda_installed || exit 1
            check_toolkit || exit 1
            check_cudnn_installed || exit 1
            check_docker_installed || exit 1
            check_docker_nvidia || exit 1
            check_ffmpeg_installed || exit 1
            check_golang_installed || exit 1
            check_python3_installed || exit 1
            check_nodejs_installed || exit 1
        elif [ "$os_name" == "Ubuntu 22 WSL" ]; then
            # Ubuntu 22 WSL
            check_toolkit || exit 1
            check_cudnn_installed || exit 1
            check_ffmpeg_installed || exit 1
            check_golang_installed || exit 1
            check_python3_installed || exit 1
            check_nodejs_installed || exit 1
        elif [ "$os_name" == "Debian 12" ]; then
            # Debian 12
            check_nvidia_driver_debian || exit 1
            check_cuda_installed_debian || exit 1
            check_toolkit || exit 1
            check_cudnn_installed || exit 1
            check_docker_installed || exit 1
            check_docker_nvidia || exit 1
            check_ffmpeg_installed || exit 1
            check_golang_installed || exit 1
            check_python3_installed || exit 1
            check_nodejs_installed || exit 1
        elif [ "$os_name" == "Debian 12 WSL" ]; then
            # Debian 12 WSL
            check_toolkit || exit 1
            check_cudnn_installed || exit 1
            check_ffmpeg_installed || exit 1
            check_golang_installed || exit 1
            check_python3_installed || exit 1
            check_nodejs_installed || exit 1
        fi
    fi
    exit 0
}

# Capture parameters
INSTALL_SIMPLE_SETUP=false
STARTED_BY_CRONJOB=false

# Display help function
show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    # echo "  --cron         Indicate the script was started by a cronjob"
    # echo "  --user         Specify the user (required with --cron)"
    echo "  --simple-setup Perform a simple setup"
    echo "  --help         Show this help message and exit"
    echo "  --check        Check all dependencies are already installed. You can use it with --simple-setup"
    exit 0
}

# Parse options
while [[ "$1" != "" ]]; do
    case "$1" in
        --cron)
            STARTED_BY_CRONJOB=true
            ;;
        --simple-setup)
            INSTALL_SIMPLE_SETUP=true
            ;;
        --user)
            shift
            if [ -z "$1" ] || [[ "$1" == -* ]]; then
                echo "Error: --user requires a valid argument"
                exit 1
            fi
            USER="$1"
            ;;
        --check)
            CHECK_DEPENDENCIES=true
            ;;
        --help)
            show_help
            ;;
        *)
            echo "Invalid option: $1"
            show_help
            ;;
    esac
    shift # Move to the next argument
done

# Validate allowed combinations
if $INSTALL_SIMPLE_SETUP && $STARTED_BY_CRONJOB; then
    echo "Invalid combination: --simple-setup cannot be combined with other options."
    exit 1
fi

# Validate required options for --cron
if [ "$STARTED_BY_CRONJOB" = true ]; then
    if [ -z "$USER" ]; then
        echo "Error: --cron requires --user to be set."
        exit 1
    fi

    SUDO_USER=$USER
fi



REBOOT_NEEDED=false

#<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
#--------------------------  wsl  ---------------------------
#<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<


#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
#--------------------------  wsl  ---------------------------
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>






#<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
#----------------------  simple-setup  ----------------------
#<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

check_reboot_configured_simple() {
    # if STARTED_BY_CRONJOB is true, then skip this task
    if [ "$STARTED_BY_CRONJOB" = true ]; then
        return 0
    else
    # if REBOOT_NEEDED is true
        if [ "$REBOOT_NEEDED" = true ]; then
            return 1
        else
            return 0
        fi
    fi
}

configure_reboot_simple() {
    REBOOT_OPTIONS="--simple-setup"
    reboot_now
}

#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
#----------------------  simple-setup  ----------------------
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>







#<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
#------------------------  ubuntu22  ------------------------
#<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

#------------------------------------------------------------
# 1. Install NVIDIA Drivers
check_nvidia_driver() {
    if command -v nvidia-smi &> /dev/null; then
        return 0  # Step is completed
    else
        return 1  # Step not completed
    fi
}

nvidia_install() {
    ubuntu-drivers autoinstall
    REBOOT_NEEDED=true
}
#------------------------------------------------------------

#------------------------------------------------------------
# 2. Install CUDA 12
check_cuda_installed() {
    if command -v nvcc &> /dev/null; then
        return 0  # Skip task
    else
        return 1  # Don't skip task
    fi
}

cuda_install() {
    # Download and install the CUDA keyring
    wget -q -O /tmp/cuda-keyring_1.0-1_all.deb https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.0-1_all.deb
    dpkg -i /tmp/cuda-keyring_1.0-1_all.deb
    rm -f /tmp/cuda-keyring_1.0-1_all.deb

    # Update and install CUDA
    apt-get update
    apt-get -y install cuda

    # Update PATH and LD_LIBRARY_PATH for the current root session
    export PATH=/usr/local/cuda-12.0/bin:$PATH
    export LD_LIBRARY_PATH=/usr/local/cuda-12.0/lib64:$LD_LIBRARY_PATH

    # Update PATH and LD_LIBRARY_PATH in root's .bashrc
    if ! grep -q '/usr/local/cuda-12.0/bin' /root/.bashrc; then
        echo 'export PATH=/usr/local/cuda-12.0/bin:$PATH' >> /root/.bashrc
    fi
    if ! grep -q '/usr/local/cuda-12.0/lib64' /root/.bashrc; then
        echo 'export LD_LIBRARY_PATH=/usr/local/cuda-12.0/lib64:$LD_LIBRARY_PATH' >> /root/.bashrc
    fi

    # Update PATH and LD_LIBRARY_PATH for the original user
    if [[ -n "$SUDO_USER" ]]; then
        ORIGINAL_USER_HOME=$(eval echo ~$SUDO_USER)
        ORIGINAL_USER_BASHRC="$ORIGINAL_USER_HOME/.bashrc"
        
        # Update .bashrc for the original user
        if ! grep -q '/usr/local/cuda-12.0/bin' "$ORIGINAL_USER_BASHRC"; then
            echo 'export PATH=/usr/local/cuda-12.0/bin:$PATH' >> "$ORIGINAL_USER_BASHRC"
        fi
        if ! grep -q '/usr/local/cuda-12.0/lib64' "$ORIGINAL_USER_BASHRC"; then
            echo 'export LD_LIBRARY_PATH=/usr/local/cuda-12.0/lib64:$LD_LIBRARY_PATH' >> "$ORIGINAL_USER_BASHRC"
        fi

        # Apply the changes to the original user's current session (if possible)
        if ps -p "$PPID" -o comm= | grep -q bash; then
            su - "$SUDO_USER" -c "export PATH=/usr/local/cuda-12.0/bin:\$PATH && export LD_LIBRARY_PATH=/usr/local/cuda-12.0/lib64:\$LD_LIBRARY_PATH"
        fi
    fi

    # Flag for reboot
    REBOOT_NEEDED=true
}
#------------------------------------------------------------

#------------------------------------------------------------
# 3. Install NVIDIA CUDA Toolkit
check_toolkit() {
    if dpkg -l | grep -qw nvidia-cuda-toolkit; then
        return 0
    else
        return 1
    fi
}

cuda_toolkit() {
    apt install -y nvidia-cuda-toolkit nvtop
}
#------------------------------------------------------------

#------------------------------------------------------------
# 4. Configure Reboot
check_reboot_configured_ubuntu22() {
    # if STARTED_BY_CRONJOB is true, then skip this task
    if [ "$STARTED_BY_CRONJOB" = true ]; then
        return 0
    else
    # if REBOOT_NEEDED is true
        if [ "$REBOOT_NEEDED" = true ]; then
            return 1
        else
            return 0
        fi
    fi
}

configure_reboot_ubuntu22() {
    REBOOT_OPTIONS=""
    reboot_now
}
#------------------------------------------------------------

#------------------------------------------------------------
# 5. Install cuDNN 8.9.7
check_cudnn_installed() {
    if dpkg -l | grep -qw libcudnn8; then
        return 0
    else
        return 1
    fi
}

install_cudnn() {
    tput clear > /dev/tty
    tput cup 0 0 > /dev/tty
    tput ed > /dev/tty
    echo -e "\e[1;31m============================== ATTENTION REQUIRED ==============================\e[0m" > /dev/tty
    echo -e "\e[1;33mNVIDIA has restricted direct downloads of libcudnn packages.\e[0m" > /dev/tty
    echo -e "\e[1;33mPlease follow the steps below:\e[0m" > /dev/tty
    echo -e "\e[1;32m1. Visit: \e[1;34mhttps://developer.nvidia.com/rdp/cudnn-archive\e[0m" > /dev/tty
    echo -e "\e[1;32m2. Download the appropriate installer for your system.\e[0m" > /dev/tty
    echo -e "\e[1;34m  (Download cuDNN v8.9.7 (December 5th, 2023), for CUDA 12.x)\e[0m" > /dev/tty
    echo -e "\e[1;32m3. Place the downloaded file in this directory.\e[0m" > /dev/tty
    echo -e "\e[1;34m  (Filename: cudnn*.deb)\e[0m" > /dev/tty
    echo -e "\e[1;33mThe script will continue automatically once the correct file is detected.\e[0m" > /dev/tty
    echo -e "\e[1;31m================================================================================\e[0m" > /dev/tty

    echo -e "\e[1;33mWaiting for a valid cudnn installer file to be saved in the current directory...\e[0m" > /dev/tty

    # Wait for a valid file to appear
    while true; do
        # Look for any file starting with 'cudnn' and ending with '.deb'
        file=$(ls cudnn*.deb 2>/dev/null | head -n 1)
        if [ -n "$file" ]; then
            # File exists, now check for stability
            previous_size=$(stat -c%s "$file")
            sleep 5
            current_size=$(stat -c%s "$file")

            if [ "$previous_size" -eq "$current_size" ]; then
                echo -e "\e[1;32mFile detected and is stable: $file. Proceeding with the installation...\e[0m" > /dev/tty
                break
            else
                echo -e "\e[1;33m$file was detected, but is still being downloaded...\e[0m" > /dev/tty
            fi
        fi
        sleep 5
        tput cuu1 > /dev/tty
    done

    # Go back to the task screen
    tput clear > /dev/tty
    update_task_status

    # Proceed with installation of the deb file
    dpkg -i "$file"
    cp /var/cudnn-local-repo-*/cudnn-local-*-keyring.gpg /usr/share/keyrings/
    apt-get update
    apt-get install -y libcudnn8 libcudnn8-dev
}
#------------------------------------------------------------

#------------------------------------------------------------
# 6. Install Docker
check_docker_installed() {
    if command -v docker &> /dev/null; then
        return 0
    else
        return 1
    fi
}

install_docker() {
    curl -sSL https://get.docker.com | sh
}
#------------------------------------------------------------

#------------------------------------------------------------
# 7. Install docker with NVIDIA support
check_docker_nvidia() {
    if dpkg -l | grep -qw nvidia-container-runtime; then
        return 0
    else
        return 1
    fi
}

install_docker_nvidia() {
    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | \
        gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
    curl -s -L "https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list" | \
        sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
        tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
    apt-get update
    apt-get install -y nvidia-container-runtime
    systemctl restart docker
}
#------------------------------------------------------------

#------------------------------------------------------------
# 8. Install ffmpeg
check_ffmpeg_installed() {
    if command -v ffmpeg &> /dev/null; then
        return 0
    else
        return 1
    fi
}

install_ffmpeg() {
    apt install -y ffmpeg
}

#------------------------------------------------------------

#------------------------------------------------------------
# 9. Install golang
check_golang_installed() {
    # Check if newer version of Go is available
    go_version=$(curl -s https://go.dev/VERSION?m=text | head -n 1 | sed 's/^go//')
    export PATH=$PATH:/usr/local/go/bin
    if command -v go &> /dev/null && [[ "$(go version)" == *"go$go_version"* ]]; then
        return 0
    else
        return 1
    fi
}

# get server architecture
ARCHITECTURE=$(dpkg --print-architecture)

install_golang() {
    echo "Starting Go installation..."

    # Get the latest Go version
    echo "Fetching the latest Go version..."
    go_version=$(curl -s https://go.dev/VERSION?m=text | head -n 1 | sed 's/^go//')
    if [[ -z "$go_version" ]]; then
        echo "Error: Failed to fetch the latest Go version." >&2
        return 1
    fi
    echo "Latest Go version is: $go_version"

    # Temporary directory for downloading
    TEMP_DIR="/temp"
    echo "Creating temporary directory at $TEMP_DIR..."
    mkdir -p "$TEMP_DIR"
    GO_ARCHIVE="${TEMP_DIR}/go${go_version}.linux-${ARCHITECTURE}.tar.gz"
    echo "Go archive will be downloaded to: $GO_ARCHIVE"

    # Download the Go binary and overwrite if it exists
    echo "Downloading Go binary..."
    wget -q -O "$GO_ARCHIVE" "https://golang.org/dl/go${go_version}.linux-${ARCHITECTURE}.tar.gz"
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to download Go binary." >&2
        return 1
    fi
    echo "Go binary downloaded successfully."

    # Remove any existing Go installation
    echo "Removing existing Go installation (if any)..."
    rm -rf /usr/local/go

    # Extract the Go archive to /usr/local
    echo "Extracting Go binary to /usr/local..."
    tar -C /usr/local -xzf "$GO_ARCHIVE"
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to extract Go binary." >&2
        return 1
    fi
    echo "Go binary extracted successfully."

    # Update PATH for the current (root) session
    echo "Updating PATH for the current session..."
    export PATH=$PATH:/usr/local/go/bin

    # Update PATH in root's .bashrc
    echo "Ensuring PATH is updated in root's .bashrc..."
    if ! grep -q '/usr/local/go/bin' /root/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
        echo "PATH updated in root's .bashrc."
    else
        echo "PATH already exists in root's .bashrc."
    fi

    # Update PATH for the original user
    if [[ -n "$SUDO_USER" ]]; then
        echo "Updating PATH for the original user: $SUDO_USER..."
        ORIGINAL_USER_HOME=$(eval echo ~$SUDO_USER)
        ORIGINAL_USER_BASHRC="$ORIGINAL_USER_HOME/.bashrc"

        if ! grep -q '/usr/local/go/bin' "$ORIGINAL_USER_BASHRC"; then
            echo 'export PATH=$PATH:/usr/local/go/bin' >> "$ORIGINAL_USER_BASHRC"
            echo "PATH updated in $SUDO_USER's .bashrc."
        else
            echo "PATH already exists in $SUDO_USER's .bashrc."
        fi

        # Update the PATH for the original user's current session (if possible)
        if ps -p "$PPID" -o comm= | grep -q bash; then
            su - "$SUDO_USER" -c "export PATH=\$PATH:/usr/local/go/bin"
            echo "PATH updated for $SUDO_USER's current session."
        fi
    else
        # For non-sudo cases, modify the current user's .bashrc
        echo "Updating PATH for the current (non-sudo) user..."
        if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
            echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
            echo "PATH updated in the current user's .bashrc."
        else
            echo "PATH already exists in the current user's .bashrc."
        fi
    fi

    echo "Go installation completed successfully."
}
#------------------------------------------------------------

#------------------------------------------------------------
# 10. Install python3
check_python3_installed() {
    if command -v python3 &> /dev/null; then
        echo "Python 3 is installed."
    else
        echo "Python 3 is not installed."
        return 1
    fi

    if command -v pip3 &> /dev/null; then
        echo "Pip 3 is installed."
    else
        echo "Pip 3 is not installed."
        return 1
    fi

    python3 -m venv --help &> /dev/null
    if [ $? -eq 0 ]; then
        echo "Python venv module is installed."
    else
        echo "Python venv module is not installed."
        return 1
    fi

    return 0
}


install_python3() {
    # Install the latest available Python 3 version and pip
    apt install -y python3 python3-pip

    # Check the installed Python 3 version dynamically
    PYTHON_VERSION=$(python3 --version | awk '{print $2}' | cut -d. -f1,2)

    # Use the detected version to install the appropriate venv package
    apt install -y python${PYTHON_VERSION}-venv
}
#------------------------------------------------------------

#------------------------------------------------------------
# 11. Install nodejs
check_nodejs_installed() {
    if [ -z "$SUDO_USER" ]; then
        SUDO_USER=$(whoami)
    fi

    USER_HOME=$(eval echo "~$SUDO_USER")
    NVM_DIR="${USER_HOME}/.nvm"

    # Check if NVM is installed for the user
    if [ -s "${NVM_DIR}/nvm.sh" ]; then
        # Load NVM and check for Node.js
        if sudo -u "$SUDO_USER" bash -c "export NVM_DIR='${NVM_DIR}'; [ -s '${NVM_DIR}/nvm.sh' ] && . '${NVM_DIR}/nvm.sh' && command -v node" &> /dev/null; then
            NODE_VERSION=$(sudo -u "$SUDO_USER" bash -c "export NVM_DIR='${NVM_DIR}'; [ -s '${NVM_DIR}/nvm.sh' ] && . '${NVM_DIR}/nvm.sh' && node -v")
            echo "Node.js ${NODE_VERSION} is already installed."
            return 0
        else
            echo "Node.js is not installed."
            return 1
        fi
    else
        echo "NVM is not installed, so Node.js cannot be found."
        return 1
    fi
}

install_nodejs() {
    USER_HOME=$(eval echo "~$SUDO_USER")
    NVM_VERSION="v0.40.1"
    NVM_INSTALL_URL="https://raw.githubusercontent.com/nvm-sh/nvm/${NVM_VERSION}/install.sh"

    # Install nvm if not already installed
    if [ ! -d "${USER_HOME}/.nvm" ]; then
        echo "Installing NVM for user $SUDO_USER..."
        sudo -u "$SUDO_USER" -H bash -c "curl -o- $NVM_INSTALL_URL | bash"
    else
        echo "NVM already seems to be installed at ${USER_HOME}/.nvm"
    fi

    NVM_INIT_LINES='
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
'

    BASHRC_FILE="${USER_HOME}/.bashrc"
    if ! sudo -u "$SUDO_USER" grep -q 'NVM_DIR' "$BASHRC_FILE"; then
        echo "Adding NVM initialization lines to ${BASHRC_FILE}"
        echo "$NVM_INIT_LINES" | sudo -u "$SUDO_USER" tee -a "$BASHRC_FILE" > /dev/null
    else
        echo "NVM initialization lines already found in ${BASHRC_FILE}"
    fi

    # Source NVM directly, then install and use Node
    sudo -u "$SUDO_USER" bash -c "export NVM_DIR=\"$USER_HOME/.nvm\" && [ -s \"$USER_HOME/.nvm/nvm.sh\" ] && \. \"$USER_HOME/.nvm/nvm.sh\" && nvm install node && nvm use node"

    echo "Node.js and NVM installed for $SUDO_USER. NVM and Node should now be available. Pls typpe 'source ~/.bashrc' to use Node.js"
}


#------------------------------------------------------------

#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
#------------------------  Ubuntu22  ------------------------
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>








#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
#------------------------  Debian12  ------------------------
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

#------------------------------------------------------------
# 1. Install NVIDIA Drivers
check_nvidia_driver_debian() {
    if lsmod | grep -q 'nvidia'; then
        return 0  # NVIDIA driver is installed
    else
        return 1  # NVIDIA driver is not installed
    fi
}

# Function to install NVIDIA driver on Debian 12
nvidia_install_debian() {
    # Install necessary packages
    apt install -y linux-headers-$(uname -r) build-essential dkms

    # Enable non-free repositories if not already enabled
    if ! grep -q 'non-free' /etc/apt/sources.list; then
        echo "Enabling non-free repositories..."
        sed -i '/^deb/s/$/ non-free/' /etc/apt/sources.list
        apt update
    fi

    # Install NVIDIA driver
    apt install -y nvidia-driver

    # Blacklist nouveau driver
    echo -e "blacklist nouveau\noptions nouveau modeset=0" | tee /etc/modprobe.d/blacklist-nouveau.conf
    update-initramfs -u

    # Inform the user to reboot
    REBOOT_NEEDED=true
}
#------------------------------------------------------------

#------------------------------------------------------------
# 2. Install CUDA 12
check_cuda_debian() {
    if command -v nvcc &> /dev/null; then
        return 0  # CUDA is installed
    else
        return 1  # CUDA is not installed
    fi
}

cuda_install_debian() {
    # Install prerequisites
    apt install -y build-essential dkms

    # Add NVIDIA package repository
    apt-key adv --fetch-keys https://developer.download.nvidia.com/compute/cuda/repos/debian12/x86_64/3bf863cc.pub
    echo "deb https://developer.download.nvidia.com/compute/cuda/repos/debian12/x86_64/ /" | tee /etc/apt/sources.list.d/cuda.list

    # Update package lists again
    apt update

    # Install CUDA
    apt install -y cuda


    # Update PATH and LD_LIBRARY_PATH for the current root session
    export PATH=/usr/local/cuda-12.0/bin:$PATH
    export LD_LIBRARY_PATH=/usr/local/cuda-12.0/lib64:$LD_LIBRARY_PATH

    # Update PATH and LD_LIBRARY_PATH in root's .bashrc
    if ! grep -q '/usr/local/cuda-12.0/bin' /root/.bashrc; then
        echo 'export PATH=/usr/local/cuda-12.0/bin:$PATH' >> /root/.bashrc
    fi
    if ! grep -q '/usr/local/cuda-12.0/lib64' /root/.bashrc; then
        echo 'export LD_LIBRARY_PATH=/usr/local/cuda-12.0/lib64:$LD_LIBRARY_PATH' >> /root/.bashrc
    fi

    # Update PATH and LD_LIBRARY_PATH for the original user
    if [[ -n "$SUDO_USER" ]]; then
        ORIGINAL_USER_HOME=$(eval echo ~$SUDO_USER)
        ORIGINAL_USER_BASHRC="$ORIGINAL_USER_HOME/.bashrc"
        
        # Update .bashrc for the original user
        if ! grep -q '/usr/local/cuda-12.0/bin' "$ORIGINAL_USER_BASHRC"; then
            echo 'export PATH=/usr/local/cuda-12.0/bin:$PATH' >> "$ORIGINAL_USER_BASHRC"
        fi
        if ! grep -q '/usr/local/cuda-12.0/lib64' "$ORIGINAL_USER_BASHRC"; then
            echo 'export LD_LIBRARY_PATH=/usr/local/cuda-12.0/lib64:$LD_LIBRARY_PATH' >> "$ORIGINAL_USER_BASHRC"
        fi

        # Apply the changes to the original user's current session (if possible)
        if ps -p "$PPID" -o comm= | grep -q bash; then
            su - "$SUDO_USER" -c "export PATH=/usr/local/cuda-12.0/bin:\$PATH && export LD_LIBRARY_PATH=/usr/local/cuda-12.0/lib64:\$LD_LIBRARY_PATH"
        fi
    fi

    # Flag for reboot
    REBOOT_NEEDED=true
}
#------------------------------------------------------------

#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
#------------------------  Debian12  ------------------------
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>












#<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
#--------------  SCRIPT STUFF: DONT TOUCH!!!  ---------------
#<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
# Detect WSL
is_wsl() {
    if grep -qi "microsoft" /proc/version || grep -q "WSL" /proc/sys/kernel/osrelease; then
        return 0
    else
        return 1
    fi
}

# Detect OS and version
get_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        if [[ "$ID" == "ubuntu" && "$VERSION_ID" == 22* ]]; then
            if is_wsl; then
                echo "Ubuntu 22 WSL"
            else
                echo "Ubuntu 22"
            fi
            return 0
        elif [[ "$ID" == "debian" && "$VERSION_ID" == 12* ]]; then
            if is_wsl; then
                echo "Debian 12 WSL"
            else
                echo "Debian 12"
            fi
            return 0
        else
            return 1
        fi
    else
        return 1
    fi
}

# Check if a screen session named "install_session" exists
if ! $STARTED_BY_CRONJOB; then
    if screen -list | grep -q $SCREEN_NAME; then
        echo "⏭️ Screen session '$SCREEN_NAME' is running. Attaching..."
        screen -r -d $SCREEN_NAME
        exit 0
    fi
fi


# Function to check for root privileges
check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo "❌ This script must be run with sudo. Please use 'sudo'." > /dev/tty
        exit 1
    fi

    if [ "$SUDO_USER" == "" ]; then
        echo "❌ This script must be run from a normal user account with sudo privileges." > /dev/tty
        exit 1
    fi
}

# Get the full path and directory of the script
FULL_PATH_OF_THIS_SCRIPT=$(readlink -f "$0")
SCRIPT_DIR=$(dirname "$FULL_PATH_OF_THIS_SCRIPT")

# Set up the log file
LOG_FILE="$SCRIPT_DIR/install.log"
touch "$LOG_FILE"
chown "$SUDO_USER:$SUDO_USER" "$LOG_FILE"

# Redirect stdout and stderr to the log file
exec >>"$LOG_FILE" 2>&1

# Function to initialize the screen
initialize_screen() {
    tput clear > /dev/tty
    tput civis > /dev/tty  # Hide cursor
    echo -e "🚀 Installation Progress:\n" > /dev/tty
    echo -e "If you want to monitor the full logs, run: 'tail -f $LOG_FILE'\n" > /dev/tty
    for ((i=0; i<${#STEPS[@]}; i++)); do
        echo -e "${STATUS_ICONS["pending"]} ${STEPS[$i]}" > /dev/tty
    done
}

# Task Management Functions

# Define statuses with icons
declare -A STATUS_ICONS=(
    ["pending"]="⏳"
    ["in_progress"]="🔄"
    ["done"]="✅"
    ["skipped"]="⏭️"
    ["error"]="❌"
)

# Array to store tasks and their statuses
declare -a TASKS
declare -A TASK_STATUSES
declare -A TASK_COMMANDS
declare -A TASK_SKIP_CHECKS

# Function to add a task
# Usage: add_task "Task Name" "Command" "Skip Function" "Pre-Check Function"
add_task() {
    local task_name="$1"
    local task_command="$2"
    local skip_check="$3"
    TASKS+=("$task_name")
    TASK_STATUSES["$task_name"]="pending"
    TASK_COMMANDS["$task_name"]="$task_command"
    TASK_SKIP_CHECKS["$task_name"]="$skip_check"
}

# Function to execute a task
run_task() {
    local task_name="$1"

    # Check if the task should be skipped
    local skip_check="${TASK_SKIP_CHECKS[$task_name]}"
    if [ -n "$skip_check" ] && $skip_check; then
        TASK_STATUSES["$task_name"]="skipped"
        update_task_status
        return
    fi

    # Execute the task
    TASK_STATUSES["$task_name"]="in_progress"
    update_task_status
    "${TASK_COMMANDS[$task_name]}"
    if [ $? -ne 0 ]; then
        TASK_STATUSES["$task_name"]="error"
        update_task_status
        exit_with_error
    else
        TASK_STATUSES["$task_name"]="done"
    fi
    update_task_status
}

# Function to initialize the task display
initialize_task_screen() {
    tput clear > /dev/tty
    tput civis > /dev/tty  # Hide cursor
    update_task_status
}

# Function to update task statuses on the screen
update_task_status() {
    tput cup 0 0 > /dev/tty
    tput ed > /dev/tty
    echo -e "🚀 Task Execution Progress:\n" > /dev/tty
    echo -e "If you want to monitor the full logs, run: 'tail -f $LOG_FILE'\n" > /dev/tty
    for task_n in "${TASKS[@]}"; do
        local status="${TASK_STATUSES[$task_n]}"
        echo -e "${STATUS_ICONS[$status]} $task_n" > /dev/tty
    done
}

# Function to exit with error
exit_with_error() {
    echo -e "❌ Error occurred during task execution. Check the logs for more details." > /dev/tty
    echo -e "\nUse 'tail $LOG_FILE' to view the log file." > /dev/tty
    echo -e "\nScript will end in 10 seconds..." > /dev/tty
    sleep 10

    # Remove LOGIN_HINT_FILE
    if [ -f "$LOGIN_HINT_FILE" ]; then
        rm -f "$LOGIN_HINT_FILE"
    fi

    tput cnorm > /dev/tty  # Show cursor
    exit 1
}

# Function to finalize the screen display
finalize_task_screen() {
    tput cnorm > /dev/tty  # Show cursor
    echo -e "\n✅ All tasks processed!" > /dev/tty
    echo -e "\nUse 'tail $LOG_FILE' to view the log file." > /dev/tty
    echo -e "\nScript will end in 10 seconds..." > /dev/tty
    sleep 10

    # Remove LOGIN_HINT_FILE
    if [ -f "$LOGIN_HINT_FILE" ]; then
        rm -f "$LOGIN_HINT_FILE"
    fi

    tput cnorm > /dev/tty  # Show cursor
}

# Main task handler
process_tasks() {
    initialize_task_screen
    for t in "${TASKS[@]}"; do
        run_task "$t"
    done
    finalize_task_screen
}

REBOOT_OPTIONS=""
ORGINAL_USER=$SUDO_USER

# Function to configure the system to reboot
reboot_now() {
    if ! grep -Fq "$FULL_PATH_OF_THIS_SCRIPT" /etc/crontab; then
        echo "@reboot root /usr/bin/screen -dmS $SCREEN_NAME /bin/bash $FULL_PATH_OF_THIS_SCRIPT --cron $REBOOT_OPTIONS --user $ORGINAL_USER" >> /etc/crontab
    fi

    # Add a login hint
    cat << EOF > "$LOGIN_HINT_FILE"
#!/bin/bash

clear

# Calculate the maximum line length for dynamic border adjustment
MAX_LENGTH=0
LINES=(
    "        🚀  INSTALLATION IN PROGRESS IN BACKGROUND  🚀        "
    "   The installation tasks are running in a 'screen' session   "
    "   named:                                                     "
    "        🖥️  SCREEN SESSION NAME: '$SCREEN_NAME'               "
    "   To monitor the installation progress, please run:          "
    "        📜  sudo su                                           "
    "        📜  screen -r -d $SCREEN_NAME                         "
    "   Once completed, a final message will be displayed here.    "
)

# Loop to determine the maximum line length
for LINE in "\${LINES[@]}"; do
    LENGTH=\${#LINE}
    if (( LENGTH > MAX_LENGTH )); then
        MAX_LENGTH=\$LENGTH
    fi
done

# Create the top border
echo -n "╔"
for (( i = 0; i < MAX_LENGTH + 4; i++ )); do
    echo -n "═"
done
echo "╗"

# Print the message within the box
for LINE in "\${LINES[@]}"; do
    LENGTH=\${#LINE}

    # Initialize extra_spaces and remove_spaces
    extra_spaces=0
    remove_spaces=0

    # Adjust extra_spaces and remove_spaces based on the content of the line
    if [[ \$LINE == *"🖥️"* ]]; then
        extra_spaces=1
        remove_spaces=0
    elif [[ \$LINE == *"📜"* ]]; then
        extra_spaces=0
        remove_spaces=1
    elif [[ \$LINE == *"🚀"* ]]; then
        extra_spaces=0
        remove_spaces=2
    else
        extra_spaces=0
        remove_spaces=0
    fi

    # Adjust the number of spaces
    num_spaces=\$(( MAX_LENGTH - LENGTH - remove_spaces ))
    if (( num_spaces < 0 )); then
        num_spaces=0
    fi

    # Create a string with num_spaces spaces
    spaces=""
    for (( i=0; i<num_spaces; i++ )); do
        spaces+=" "
    done

    # Create a string with extra_spaces spaces
    extra_spaces_str=""
    for (( i=0; i<extra_spaces; i++ )); do
        extra_spaces_str+=" "
    done

    printf "║  %s%s%s  ║\\n" "\$LINE" "\$spaces" "\$extra_spaces_str"
done

# Create the bottom border
echo -n "╚"
for (( i = 0; i < MAX_LENGTH + 4; i++ )); do
    echo -n "═"
done
echo "╝"

EOF

    chmod +x "$LOGIN_HINT_FILE"
    sleep 5
    reboot now
}


#------------------------------------------------------------
# 0. Update System Packages
update() {
    apt update
}
#------------------------------------------------------------

#------------------------------------------------------------
# 0. Install nessecary packages
check_nessecary_packages_installed() {
    if dpkg -l | grep -qw wget && dpkg -l | grep -qw curl && dpkg -l | grep -qw dpkg && dpkg -l | grep -qw screen && dpkg -l | grep -qw make && dpkg -l | grep -qw grep; then
        return 0
    else
        return 1
    fi
}

install_nessecary_packages() {
    apt install wget curl dpkg screen make grep -y
}

#------------------------------------------------------------

if [ "$CHECK_DEPENDENCIES" = true ]; then
    check_dependencies
    exit 0
fi

# Check for root access
check_root

add_task "Update System Packages" update ""  # No skip function
add_task "Install nessecary packages" install_nessecary_packages check_nessecary_packages_installed  # Skip if drivers are installed

# Run the main function
main

#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
#--------------  SCRIPT STUFF: DONT TOUCH!!!  ---------------
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>