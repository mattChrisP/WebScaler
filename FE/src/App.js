import React, { useState, useEffect } from 'react';
import './App.css';
import { v4 as uuidv4 } from 'uuid';

function App() {
  const [selectedImage, setSelectedImage] = useState(null);
  const [processedImage, setProcessedImage] = useState(null);
  const [loading, setLoading] = useState(false);
  const [uniqueId, setUniqueId] = useState(null);
  const [intervalId, setIntervalId] = useState(null);

  const handleImageChange = async (e) => {
    const file = e.target.files[0];
    if (file && file.type.startsWith('image/')) {
      const reader = new FileReader();
      reader.onloadend = () => {
        setSelectedImage(reader.result);
        uploadImage(file);
      };
      reader.readAsDataURL(file);
    } else {
      setSelectedImage(null);
      alert('Please select a valid image.');
    }
  };

  const fetchProcessedImg = async (idToFetch) => {
    try {
      const response = await fetch(`http://localhost:8080/get-upscaled-image?uniqueId=${idToFetch}`);
      if (response.ok) {
        const data = await response.blob();
        const imageUrl = URL.createObjectURL(data);
        setProcessedImage(imageUrl);
        if (intervalId) {
          clearInterval(intervalId);
        }
        setLoading(false);
      }
    } catch (error) {
      console.error("Failed fetching processed image:", error);
    }
  };

  const uploadImage = async (image) => {
    setLoading(true);
    const id = uuidv4();

    const formData = new FormData();
    formData.append('image', image);
    formData.append('id', id);

    try {
      const response = await fetch('http://localhost:8080/upload', {
        method: 'POST',
        body: formData
      });

      if (!response.ok) {
        throw new Error('Failed to upload image.');
      }

      const pollingId = setInterval(() => fetchProcessedImg(id), 2000);
      setIntervalId(pollingId);
      setUniqueId(id);
    } catch (error) {
      console.error("Error:", error);
      setLoading(false);
    }
  };

  useEffect(() => {
    return () => {
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }, [intervalId]);

  return (
    <div className="App">
      <header className="App-header">
        <h1>Web Scaler</h1>
        <input type="file" onChange={handleImageChange} />

        {loading && <p>Loading...</p>}

        <div style={{ display: 'flex', justifyContent: 'space-between', width: '650px', marginTop: '20px' }}>
          {selectedImage && <img src={selectedImage} alt="Chosen" style={{ width: '300px' }} />}
          {processedImage && <img src={processedImage} alt="Processed" style={{ width: '300px' }} />}
        </div>
      </header>
    </div>
  );
}

export default App;


