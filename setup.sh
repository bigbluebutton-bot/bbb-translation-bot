#!/bin/bash

SCREEN_NAME="install_session"
LOGIN_HINT_FILE="/etc/profile.d/install_hint.sh"

# Main execution: Define tasks and run the main function
main() {
    if $INSTALL_ON_UBUNTU22; then
        add_task "Update System Packages" update ""  # No skip function
        add_task "Install NVIDIA Drivers" nvidia_install check_nvidia_driver  # Skip if drivers are installed
        add_task "Install NVIDIA CUDA" cuda_install check_cuda_installed  # Skip if CUDA is installed
        add_task "Install NVIDIA CUDA Toolkit" cuda_toolkit check_toolkit  # Skip if toolkit is installed
        add_task "Reboot" configure_reboot_ubuntu22 check_reboot_configured_ubuntu22  # Skip if reboot is needed
        add_task "Install NVIDIA cuDNN 8.9.7" install_cudnn check_cudnn_installed  # Skip if cuDNN is installed
        add_task "Install Docker" install_docker check_docker_installed  # Skip if Docker is installed
        add_task "Install Docker with NVIDIA support" install_docker_nvidia check_docker_nvidia  # Skip if Docker with NVIDIA support is installed
        add_task "Install golang" install_golang check_golang_installed  # Skip if golang is installed
        add_task "Install python3" install_python3 check_python3_installed  # Skip if python3 is installed
        add_task "Install nodejs" install_nodejs check_nodejs_installed  # Skip if nodejs is installed
    elif $INSTALL_SIMPLE_SETUP; then
        add_task "Update System Packages" update ""  # No skip function
        add_task "Install NVIDIA Drivers" nvidia_install check_nvidia_driver  # Skip if drivers are installed
        add_task "Install NVIDIA CUDA" cuda_install check_cuda_installed  # Skip if CUDA is installed
        add_task "Install NVIDIA CUDA Toolkit" cuda_toolkit check_toolkit  # Skip if toolkit is installed
        add_task "Reboot" configure_reboot_simple check_reboot_configured_simple  # Skip if reboot is needed
        add_task "Install Docker" install_docker check_docker_installed  # Skip if Docker is installed
        add_task "Install Docker with NVIDIA support" install_docker_nvidia check_docker_nvidia  # Skip if Docker with NVIDIA support is installed
    elif $INSTALL_ON_WSL; then
        add_task "Update System Packages" update ""  # No skip function
        add_task "Install NVIDIA CUDA Toolkit" cuda_toolkit check_toolkit  # Skip if toolkit is installed
        add_task "Install golang" install_golang check_golang_installed  # Skip if golang is installed
        add_task "Install python3" install_python3 check_python3_installed  # Skip if python3 is installed
        add_task "Install nodejs" install_nodejs check_nodejs_installed  # Skip if nodejs is installed
    fi

    # Process task-specific logic
    process_tasks
}

CHECK_DEPENDENCIES=false
check_dependencies() {
    if $INSTALL_ON_UBUNTU22; then
        check_nvidia_driver || exit 1
        check_cuda_installed || exit 1
        check_toolkit || exit 1
        check_cudnn_installed || exit 1
        check_docker_installed || exit 1
        check_docker_nvidia || exit 1
        check_golang_installed || exit 1
        check_python3_installed || exit 1
        check_nodejs_installed || exit 1
    elif $INSTALL_SIMPLE_SETUP; then
        check_nvidia_driver || exit 1
        check_cuda_installed || exit 1
        check_toolkit || exit 1
        check_docker_installed || exit 1
        check_docker_nvidia || exit 1
    elif $INSTALL_ON_WSL; then
        check_toolkit || exit 1
        check_golang_installed || exit 1
        check_python3_installed || exit 1
        check_nodejs_installed || exit 1
    fi
    exit 0
}

# Capture parameters
INSTALL_ON_UBUNTU22=false
INSTALL_ON_WSL=false
INSTALL_SIMPLE_SETUP=false
STARTED_BY_CRONJOB=false

# Display help function
show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    # echo "  --cron         Indicate the script was started by a cronjob"
    # echo "  --path         Specify the folder path (required with --cron)"
    # echo "  --user         Specify the user (required with --cron)"
    echo "  --simple-setup Perform a simple setup"
    echo "  --ubuntu22     Install on Ubuntu 22.04"
    echo "  --wsl          Install on Windows Subsystem for Linux with Ubuntu 22.04"
    echo "  --help         Show this help message and exit"
    echo "  --check        Check all dependencies are already installed. Use with --ubuntu22, --wsl, or --simple-setup"
    exit 0
}

# Check if no arguments were provided
if [ "$#" -eq 0 ]; then
    show_help
fi

# Parse options
while [[ "$1" != "" ]]; do
    case "$1" in
        --cron)
            STARTED_BY_CRONJOB=true
            ;;
        --ubuntu22)
            INSTALL_ON_UBUNTU22=true
            ;;
        --wsl)
            INSTALL_ON_WSL=true
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
        --path)
            shift
            if [ -z "$1" ] || [[ "$1" == -* ]]; then
                echo "Error: --path requires a valid folder path"
                exit 1
            fi
            PATH_TO_FOLDER="$1"
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
if $INSTALL_SIMPLE_SETUP && ($INSTALL_ON_UBUNTU22 || $INSTALL_ON_WSL || $STARTED_BY_CRONJOB); then
    echo "Invalid combination: --simple-setup cannot be combined with other options."
    exit 1
fi

if $INSTALL_ON_UBUNTU22 && $INSTALL_ON_WSL; then
    echo "Invalid combination: --ubuntu22 and --wsl cannot be used together."
    exit 1
fi

if $STARTED_BY_CRONJOB && ! $INSTALL_ON_UBUNTU22; then
    echo "Invalid combination: --cron must be used with --ubuntu22."
    exit 1
fi

# Validate required options for --cron
if [ "$STARTED_BY_CRONJOB" = true ]; then
    if [ -z "$PATH_TO_FOLDER" ]; then
        echo "Error: --cron requires --path to be set."
        exit 1
    fi
    if [ -z "$USER" ]; then
        echo "Error: --cron requires --user to be set."
        exit 1
    fi
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
# 1. Update System Packages
update() {
    apt update
}
#------------------------------------------------------------

#------------------------------------------------------------
# 2. Install NVIDIA Drivers
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
# 3. Install CUDA 12
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
# 4. Install NVIDIA CUDA Toolkit
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
# 5. Configure Reboot
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
    REBOOT_OPTIONS="--ubuntu22"
    reboot_now
}
#------------------------------------------------------------

#------------------------------------------------------------
# 6. Install cuDNN 8.9.7
check_cudnn_installed() {
    if dpkg -l | grep -qw libcudnn8; then
        return 0
    else
        return 1
    fi
}

install_cudnn() {
    wget -q https://developer.download.nvidia.com/compute/cudnn/secure/8.9.7/local_installers/12.x/cudnn-local-repo-ubuntu2204-8.9.7.29_1.0-1_amd64.deb
    dpkg -i cudnn-local-repo-ubuntu2204-8.9.7.29_1.0-1_amd64.deb
    rm -f cudnn-local-repo-ubuntu2204-8.9.7.29_1.0-1_amd64.deb
    apt-key add /var/cudnn-local-repo-*/7fa2af80.pub
    apt-get update
    apt-get install -y libcudnn8 libcudnn8-dev
}
#------------------------------------------------------------

#------------------------------------------------------------
# 7. Install Docker
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
# 8. Install docker with NVIDIA support
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
    # Get the latest Go version
    go_version=$(curl -s https://go.dev/VERSION?m=text | head -n 1 | sed 's/^go//')
    
    # Temporary directory for downloading
    TEMP_DIR="/temp"
    mkdir -p "$TEMP_DIR"
    GO_ARCHIVE="${TEMP_DIR}/go${go_version}.linux-${ARCHITECTURE}.tar.gz"
    
    # Download the Go binary and overwrite if it exists
    wget -q -O "$GO_ARCHIVE" "https://golang.org/dl/go${go_version}.linux-${ARCHITECTURE}.tar.gz"
    
    # Remove any existing Go installation
    rm -rf /usr/local/go
    
    # Extract the Go archive to /usr/local
    tar -C /usr/local -xzf "$GO_ARCHIVE"
    
    # Update PATH for the current (root) session
    export PATH=$PATH:/usr/local/go/bin
    
    # Update PATH in root's .bashrc
    if ! grep -q '/usr/local/go/bin' /root/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
    fi

    # Update PATH for the original user
    if [[ -n "$SUDO_USER" ]]; then
        ORIGINAL_USER_HOME=$(eval echo ~$SUDO_USER)
        ORIGINAL_USER_BASHRC="$ORIGINAL_USER_HOME/.bashrc"
        
        # Add Go binary to the original user's PATH permanently
        if ! grep -q '/usr/local/go/bin' "$ORIGINAL_USER_BASHRC"; then
            echo 'export PATH=$PATH:/usr/local/go/bin' >> "$ORIGINAL_USER_BASHRC"
        fi

        # Update the PATH for the original user's current session (if possible)
        if ps -p "$PPID" -o comm= | grep -q bash; then
            su - "$SUDO_USER" -c "export PATH=\$PATH:/usr/local/go/bin"
        fi
    else
        # For non-sudo cases, modify the current user's .bashrc
        if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
            echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        fi
    fi
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




















#<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
#--------------  SCRIPT STUFF: DONT TOUCH!!!  ---------------
#<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
# Check if a screen session named "install_session" exists
if screen -list | grep -q $SCREEN_NAME; then
  echo "⏭️ Screen session '$SCREEN_NAME' is running. Attaching..."
  screen -r $SCREEN_NAME
  exit 0
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

# Check for root access
check_root

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
        echo "@reboot root /usr/bin/screen -dmS $SCREEN_NAME /bin/bash $FULL_PATH_OF_THIS_SCRIPT --cron $REBOOT_OPTIONS --path $SCRIPT_DIR --user $ORGINAL_USER" >> /etc/crontab
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
    "        📜  screen -r $SCREEN_NAME                            "
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
    reboot now
}

if [ "$CHECK_DEPENDENCIES" = true ]; then
    check_dependencies
    exit 0
fi
# Run the main function
main

#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
#--------------  SCRIPT STUFF: DONT TOUCH!!!  ---------------
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>