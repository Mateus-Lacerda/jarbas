import pyttsx3


engine = pyttsx3.init()
voices = engine.getProperty('voices')
with open('voices.txt', 'w') as f:
  for voice in voices:
    f.write(str(voice) + '\n')


engine.runAndWait()
