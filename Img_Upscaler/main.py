import cv2
import logging


from flask import Flask, jsonify, request

app = Flask(__name__)

app.logger.setLevel(logging.INFO)


@app.route('/')
def home():
    return "Empty Page"

@app.route('/upscale', methods=['POST'])
def upscale():
    app.logger.info("I am reached")
    image_file = request.files.get("image", None)
    return "temp", 200

    # return send_file(output_path, mimetype='image/png')
    # Load the model
    sr = cv2.dnn_superres.DnnSuperResImpl_create()

    # Path to the .pb model file
    path = "models/ESPCN_x4.pb"

    # Read the desired model
    sr.readModel(path)

    # Set the desired model and scale to get correct pre- and post-processing
    sr.setModel("espcn", 4)

    # Read the image
    image = cv2.imread(image_file)

    # Use the model to upscale the image
    result = sr.upsample(image)

    # Save the image
    cv2.imwrite('upscaled_image.png', result)
    
    return jsonify({'result' : result})

if __name__ == '__main__':
    app.run(host="0.0.0.0", port=5000)

