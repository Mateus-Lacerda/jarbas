# Personal Assistant Jarbas

Jarbas is a personal assistant built with Python that can interact with you via voice commands or text input. This assistant can perform web searches, summarize texts, respond to various commands and generate, execute and debug code using a local language model.

## Features

- Voice and text command modes
- Text-to-speech (TTS) with configurable voice
- Speech recognition with configurable language
- Web search and summarization using Brave Search
- Customizable configurations via a JSON file

## Setup

### Prerequisites

- Python <3.9 and <3.12
- Required Python packages:
  - `requests`
  - `pyttsx3`
  - `demoji`
  - `beautifulsoup4`
  - `speechrecognition`
  - `sounddevice`
  - `open-interpreter`

### Installation

1. Clone the repository or download the source code.
2. Create a virtual environment and activate it:
    ```sh
    python -m venv venv
    source venv/bin/activate  # On Windows use `venv\Scripts\activate`
    ```
3. Install the required packages:
    ```sh
    pip install -r requirements.txt
    ```
4. Ensure you have the `voices.txt` file in the same directory as the script. This file should list the available voices. \
   
\* Optional: If you don't want to pay for a llm model, you might consider installing Ollama, or LMStudio, to keep Jarbas private and local 

### Configuration

Create or update the `config.json` file in the same directory as the script. Below is an example configuration:

```json
{
    "bot_name": "Jarbas",
    "user_name": "Master",
    "voice": "English (America)",
    "speech_rate": 160,
    "model": "openai/llama3",
    "api_key": "ollama",
    "api_base": "http://localhost:11434/v1",
    "context_window": 3000,
    "max_tokens": 300,
    "command_mode": "voice",
    "max_speak_length": 200,
    "language": "en-US"
}
```

### Usage

1. Ensure your configuration file (config.json) is set up correctly.
Run the script to start the assistant:
```sh
python jarbas.py
```
Interact with the assistant using voice commands or text input based on your configuration.

### Voice Configuration
The voices.txt file should list the available voices in the following format:
```php
<Voice id=English (America)
      name=English (America)
      languages=[b'\x02en-us']
      gender=male
      age=None>
```      
Make sure the voice specified in the config.json ("voice": "English (America)") matches one of the voices listed in voices.txt.

### Multilanguage support (Upcoming)
...

### Commands
- Voice Command Mode: Speak your command clearly after the assistant prompts you.
- Text Command Mode: Type your command into the terminal.
### Example Commands
- Web Search: "search for Bitcoin"
- General Questions: "What's the weather like today?"
- Exit the Assistant: "exit" or "quit"
### Handling Long Responses
If the response is too long, the assistant will ask if you want it to be read out loud. You can respond with "yes" or "no".

### Troubleshooting
- Voice Not Found: Ensure the voice name in config.json matches exactly with one listed in voices.txt.
- Speech Recognition Issues: Check your microphone settings and ensure it is configured correctly.
- Dependencies: Ensure all required Python packages are installed.

### Contributing
Feel free to fork this repository and submit pull requests. For major changes, please open an issue first to discuss what you would like to change.


