import React, { useState, useEffect } from "react";
import "./App.css"
import FileUpload from "./components/FileUpload";
import FileList from "./components/FileList";

function App() {
    const [files, setFiles] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    const fetchFiles = async () => {
        setLoading(true);
        try {
            const response = await fetch("http://localhost:8080/files");
            if (!response.ok) {
                throw new Error("Error fetching files");
            }
            const data = (await response.json()) || [];
            setFiles(data);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    // Fetch files on component mount
    useEffect(() => {
        fetchFiles();
    }, []);

    return (
        <div className="App">
            {/* Pass fetchFiles as a callback so FileUpload can trigger an update */}
            <FileUpload onUploadSuccess={fetchFiles} />
            <FileList files={files} loading={loading} error={error} refreshFiles={fetchFiles} />
        </div>
    );
}

export default App;

