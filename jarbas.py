import warnings
import json
import requests
import pyttsx3
import demoji
import re
import threading
from bs4 import BeautifulSoup
import speech_recognition as sr
import sounddevice as sd
from interpreter import interpreter

# Suppress all warnings
warnings.filterwarnings("ignore")


class Assistant:
    def __init__(self):
        # Load configuration from JSON file
        with open('config.json', 'r') as config_file:
            config = json.load(config_file)

        self.bot_name = config.get('bot_name', 'Assistant')
        self.user_name = config.get('user_name', 'User')
        self.voice_id = config.get('voice', 'English (America)')
        self.speech_rate = config.get('speech_rate', 160)
        model = config.get('model', 'openai/llama3')
        api_key = config.get('api_key', 'ollama')
        api_base = config.get('api_base', 'http://localhost:11434/v1')
        context_window = config.get('context_window', 3000)
        max_tokens = config.get('max_tokens', 300)
        self.command_mode = config.get('command_mode', 'voice')  # Pode ser "text" ou "voice"
        self.max_speak_length = config.get('max_speak_length', 200)  # Limite de comprimento do texto falado
        self.language = config.get('language', 'en-US')  # Código de idioma para reconhecimento de voz e síntese de voz

        # Initialize pyttsx3
        self.engine = pyttsx3.init()

        # Set properties for the TTS engine
        voices = self.engine.getProperty('voices')
        selected_voice = None
        for voice in voices:
            if self.voice_id in voice.id:
                selected_voice = voice
                break

        if selected_voice:
            self.engine.setProperty('voice', selected_voice.id)
        else:
            print(f"Voice with id '{self.voice_id}' not found. Using default voice.")
            self.engine.setProperty('voice', voices[0].id)  # Fallback to the first available voice


        self.engine.setProperty('rate', self.speech_rate)  # Adjust the speed as necessary

        # Initialize speech recognition
        self.recognizer = sr.Recognizer()
        self.mic = sr.Microphone()

        # Lock for pyttsx3 engine to avoid race conditions
        self.tts_lock = threading.Lock()

        self.interpreter = interpreter

        # Disabling online features
        self.interpreter.offline = True

        # Configuring the interpreter to use the local Ollama server with model details from the config
        self.interpreter.llm.model = model
        self.interpreter.llm.api_key = api_key
        self.interpreter.llm.api_base = api_base
        self.interpreter.llm.context_window = context_window
        self.interpreter.llm.max_tokens = max_tokens

        # Customize the system message
        self.interpreter.system_message += f"Your name is {self.bot_name}, you are a personal assistant for me {self.user_name}. "
        self.interpreter.system_message += "You are a personal assistant, you are here to help me with my daily tasks. "
        self.interpreter.system_message += "You are running on a Ubuntu machine, you are a Python program."
        self.interpreter.system_message += f"You speak {self.language} and you are using the {model} model for language processing. "

        # Enable spoken messages
        self.interpreter.speak_messages = True

    def run_tts(self, text):
        with self.tts_lock:
            self.engine.say(text)
            self.engine.runAndWait()

    # Function to handle text output and convert it to speech using pyttsx3
    def speak_message(self, message):
        if isinstance(message, list):
            message = " ".join([str(item) for item in message])

        # Remove emojis from the message
        message = demoji.replace(message, "")
        # Remove code blocks from the message
        message = re.sub(r'```.*?```', '', message, flags=re.DOTALL)
        # Remove any other code-like text (inline code)
        message = re.sub(r'`[^`]*`', '', message)
        # Remove special characters from the message
        message = message.replace("#", "").replace("*", "").replace("_", "").replace("-", "").replace("~", "")

        if len(message) > self.max_speak_length:
            print("The message is too long to be spoken directly.")
            if self.command_mode == 'voice':
                self.run_tts("The message is quite long. Do you want me to read it?")
                response = None
                while response is None:
                    response = self.recognize_speech()
                response = response.strip().lower()
                if response in ["yes", "sure", "go ahead"]:
                    self.run_tts("Okay, here it is:")
                    threading.Thread(target=self.run_tts, args=(message,)).start()
                else:
                    print("Not speaking the long message.")
            else:
                user_input = input("The message is quite long. Do you want me to read it? (yes/no): ").strip().lower()
                if user_input == "yes":
                    print("Okay, here it is:")
                    threading.Thread(target=self.run_tts, args=(message,)).start()
                else:
                    print("Not speaking the long message.")
        else:
            # Use pyttsx3 in a separate thread to avoid blocking
            threading.Thread(target=self.run_tts, args=(message,)).start()

    # Function to perform a web search using Brave Search
    def brave_search(self, query):
        search_url = "https://search.brave.com/search"
        params = {"q": query}
        response = requests.get(search_url, params=params)
        if response.status_code == 200:
            soup = BeautifulSoup(response.text, 'html.parser')

            # Extract only meaningful text
            results = []
            for result in soup.find_all(['h2', 'h3', 'p']):
                text = result.get_text().strip()
                if text:
                    results.append(text)
            
            # Join the results and prepare for Ollama
            text_for_ollama = "\n".join(results[:10])  # Adjust the number of results if needed

            # Use Ollama to filter and summarize the results
            response = self.interpreter.chat("summarize the following text: " + text_for_ollama)
            if isinstance(response, list):
                response = " ".join(item['content'] for item in response if isinstance(item, dict) and 'content' in item)

            return response
        else:
            return "Sorry, I couldn't perform the search. Please try again later."

    # Function to recognize speech and return the text
    def recognize_speech(self):
        with self.mic as source:
            self.recognizer.adjust_for_ambient_noise(source)
            print("Listening for a command...")
            audio = self.recognizer.listen(source)
        try:
            command = self.recognizer.recognize_google(audio, language=self.language)
            print(f"Recognized command: {command}")
            return command
        except sr.UnknownValueError:
            print("Could not understand the audio")
            return None
        except sr.RequestError:
            print("Could not request results; check your network connection")
            return None

    # Function to handle text input
    @classmethod
    def text_command_input(cls):
        return input("Enter command: ").strip().lower()

    # Custom function to start the chat and speak the responses
    def start_chat(self):
        self.speak_message(f"Hello, {self.user_name}. {self.bot_name} here. What's up?")
        print(f"Starting the chat with {self.bot_name} in {self.command_mode} mode...")
        print("You can start talking to the assistant now. Say 'exit' or 'quit' to stop the conversation.")
        print("You can also say 'search for' followed by a query to perform a web search.")
        print("You can switch to text mode by saying 'switch to text mode'.")
        while True:
            if self.command_mode == "voice":
                command = self.recognize_speech()
                if command is None:
                    continue
                command = command.lower().strip()
            else:
                command = __class__.text_command_input()

            if command in ["exit", "quit"]:
                self.speak_message(f"Goodbye, {self.user_name}.")
                return
            if command.startswith("search"):
                query = command[10:].strip()
                search_result = self.brave_search(query)
                self.speak_message(search_result)
            elif command in ["switch to text mode", "text mode", "switch to text"]:
                self.command_mode = "text"
                self.speak_message("Switched to text mode.")
            elif command in ["switch to voice mode", "voice mode", "switch to voice"]:
                self.command_mode = "voice"
                self.speak_message("Switched to voice mode.")
            else:
                response = self.interpreter.chat(command)
                if isinstance(response, list):
                    response = " ".join(item['content'] for item in response if isinstance(item, dict) and 'content' in item)
                self.speak_message(response)

# Start the custom chat function
assistant = Assistant()
assistant.start_chat()
