# Spy Bot

Spy Bot is a Telegram bot designed for monitoring and logging user activities within a chat. It captures events such as messages, edits, deletions, and other interactions, storing them for analysis and review.

## Features
- **Message Tracking**: Logs all sent messages in the chat.
- **Edit Detection**: Monitors and records message edits.
- **Deletion Logging**: Keeps track of deleted messages.

## Prerequisites
- **Docker & Docker Compose**: Required for containerized deployment.

## Installation

1. **Clone the Repository**:

    ```bash
    git clone https://github.com/sudora1n/spy-bot.git
    ```


2. **Navigate to the Project Directory**:

    ```bash
    cd spy-bot
    ```


3. **Prepare**:

    Run
    ```bash
    ./prepare.sh
    ```
    \
    and configure the necessary environment variables in .env

## Docker Deployment

1. **Build and Start Containers**:

    ```bash
    docker compose up --build
    ```


2. **Stop Containers**:

    ```bash
    docker compose down
    ```


## Configuration

The application can be configured using the `.env` file. Ensure all required variables are set before running the application.

## License

This project is licensed under the MIT License. See the LICENSE file for details.
