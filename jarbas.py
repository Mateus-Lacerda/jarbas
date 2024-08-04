import warnings
import os
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

# Load configuration from JSON file
with open('config.json', 'r') as config_file:
    config = json.load(config_file)

bot_name = config.get('bot_name', 'Assistant')
user_name = config.get('user_name', 'User')
voice_id = config.get('voice', 'English (America)')
speech_rate = config.get('speech_rate', 160)
model = config.get('model', 'openai/llama3')
api_key = config.get('api_key', 'ollama')
api_base = config.get('api_base', 'http://localhost:11434/v1')
context_window = config.get('context_window', 3000)
max_tokens = config.get('max_tokens', 300)
command_mode = config.get('command_mode', 'voice')  # Pode ser "text" ou "voice"
max_speak_length = config.get('max_speak_length', 200)  # Limite de comprimento do texto falado
language = config.get('language', 'en-US')  # Código de idioma para reconhecimento de voz e síntese de voz

# Initialize pyttsx3
engine = pyttsx3.init()

# Set properties for the TTS engine
voices = engine.getProperty('voices')
selected_voice = None
for voice in voices:
    if voice_id in voice.id:
        selected_voice = voice
        break

if selected_voice:
    engine.setProperty('voice', selected_voice.id)
else:
    print(f"Voice with id '{voice_id}' not found. Using default voice.")
    engine.setProperty('voice', voices[0].id)  # Fallback to the first available voice


engine.setProperty('rate', speech_rate)  # Adjust the speed as necessary

# Initialize speech recognition
recognizer = sr.Recognizer()
mic = sr.Microphone()

# Lock for pyttsx3 engine to avoid race conditions
tts_lock = threading.Lock()

# Function to handle text output and convert it to speech using pyttsx3
def speak_message(message):
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

    def run_tts(text):
        with tts_lock:
            engine.say(text)
            engine.runAndWait()

    if len(message) > max_speak_length:
        print("The message is too long to be spoken directly.")
        if command_mode == 'voice':
            run_tts("The message is quite long. Do you want me to read it?")
            response = recognize_speech().lower()
            if response in ["yes", "sure", "go ahead"]:
                run_tts("Okay, here it is:")
                threading.Thread(target=run_tts, args=(message,)).start()
            else:
                print("Not speaking the long message.")
        else:
            user_input = input("The message is quite long. Do you want me to read it? (yes/no): ").strip().lower()
            if user_input == "yes":
                print("Okay, here it is:")
                threading.Thread(target=run_tts, args=(message,)).start()
            else:
                print("Not speaking the long message.")
    else:
        # Use pyttsx3 in a separate thread to avoid blocking
        threading.Thread(target=run_tts, args=(message,)).start()

# Function to perform a web search using Brave Search
def brave_search(query):
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
        response = interpreter.chat("summarize the following text: " + text_for_ollama)
        if isinstance(response, list):
            response = " ".join(item['content'] for item in response if isinstance(item, dict) and 'content' in item)

        return response
    else:
        return "Sorry, I couldn't perform the search. Please try again later."

# Disabling online features
interpreter.offline = True

# Configuring the interpreter to use the local Ollama server with model details from the config
interpreter.llm.model = model
interpreter.llm.api_key = api_key
interpreter.llm.api_base = api_base
interpreter.llm.context_window = context_window
interpreter.llm.max_tokens = max_tokens

# Customize the system message
interpreter.system_message += f"Your name is {bot_name}, you are a personal assistant for me {user_name}. "
interpreter.system_message += "You are a personal assistant, you are here to help me with my daily tasks. "
interpreter.system_message += "You are running on a Ubuntu machine, you are a Python program."
interpreter.system_message += f"You speak {language} and you are using the {model} model for language processing. "

# Enable spoken messages
interpreter.speak_messages = True

# Function to recognize speech and return the text
def recognize_speech():
    with mic as source:
        recognizer.adjust_for_ambient_noise(source)
        print("Listening for a command...")
        audio = recognizer.listen(source)
    try:
        command = recognizer.recognize_google(audio, language=language)
        print(f"Recognized command: {command}")
        return command
    except sr.UnknownValueError:
        print("Could not understand the audio")
        return None
    except sr.RequestError:
        print("Could not request results; check your network connection")
        return None

# Function to handle text input
def text_command_input():
    return input("Enter command: ").strip().lower()

# Custom function to start the chat and speak the responses
def start_chat():
    speak_message(f"Hello, {user_name}. {bot_name} here. What's up?")
    while True:
        if command_mode == "voice":
            command = recognize_speech()
            if command is None:
                continue
            command = command.lower()
        else:
            command = text_command_input()

        if command in ["exit", "quit"]:
            speak_message(f"Goodbye, {user_name}.")
            return
        if command.startswith("search for"):
            query = command[10:].strip()
            search_result = brave_search(query)
            speak_message(search_result)
        else:
            response = interpreter.chat(command)
            if isinstance(response, list):
                response = " ".join(item['content'] for item in response if isinstance(item, dict) and 'content' in item)
            speak_message(response)

# Start the custom chat function
start_chat()
