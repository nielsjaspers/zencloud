import React, { useState } from "react";

export default function FileUpload({ onUploadSuccess }) {
    const [selectedFile, setSelectedFile] = useState(null);
    const [uploadResult, setUploadResult] = useState(null);

    const handleFileChange = (e) => {
        setSelectedFile(e.target.files[0]);
    };

    const handleUpload = async () => {
        if (!selectedFile) return;
        const formData = new FormData();
        formData.append("file", selectedFile);

        try {
            const response = await fetch("http://localhost:8080/upload", {
                method: "POST",
                body: formData,
            });
            const data = await response.json();
            setUploadResult(data);
            // Clear the file input (optional)
            setSelectedFile(null);
            // Call the upload success callback to refresh the file list
            if (onUploadSuccess) {
                onUploadSuccess();
            }
        } catch (err) {
            console.error("Error uploading file", err);
        }
    };

    return (
        <div>
            <h3>Upload File</h3>
            <input type="file" onChange={handleFileChange} />
            <button onClick={handleUpload}>Upload</button>
            {uploadResult && (
                <div>
                    <p>Upload Successful!</p>
                    <p>ID: {uploadResult.id}</p>
                </div>
            )}
        </div>
    );
}

