import json
import urllib.request

url = "https://huggingface.co/microsoft/BitNet-b1.58-2B-4T/raw/main/tokenizer.json"
response = urllib.request.urlopen(url)
data = json.loads(response.read())

for added in data['added_tokens']:
    if 'eot_id' in added['content']:
        print("eot_id info:", added)
