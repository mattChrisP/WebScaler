import cv2
import logging

import numpy as np
from PIL import Image
import io


from flask import Flask, jsonify, request, send_file, Response

app = Flask(__name__)

app.logger.setLevel(logging.INFO)


@app.route('/')
def home():
    return "Empty Page"


@app.route('/upscale', methods=['POST'])
def upscale():
    image_file = request.files.get("image", None)
    
    if not image_file:
        return jsonify({'error': 'No image provided'}), 400

    unique_id = request.form.get("uniqueId", None)
    stri = "Error: do not receive unique id"
    app.logger.info(f"{unique_id if unique_id != None else stri}")

    # Convert the uploaded image to a format suitable for OpenCV
    image_stream = Image.open(image_file.stream)
    image_np = np.array(image_stream)

    # Convert from RGB to BGR
    image_np = cv2.cvtColor(image_np, cv2.COLOR_RGB2BGR)

    # Load the model
    sr = cv2.dnn_superres.DnnSuperResImpl_create()

    # Path to the .pb model file
    path = "models/ESPCN_x4.pb"

    # Read the desired model
    sr.readModel(path)

    # Set the desired model and scale to get correct pre- and post-processing
    sr.setModel("espcn", 4)

    # Use the model to upscale the image
    result = sr.upsample(image_np)

    # Save the image
    output_path = f'{unique_id}-upscaled.png'
    cv2.imwrite(output_path, result)

    # Convert the upscaled image to a byte stream and return it
    _, buffer = cv2.imencode('.png', result)
    return Response(buffer.tobytes(), content_type='image/png')

if __name__ == '__main__':
    app.run(host="0.0.0.0", port=5000)

