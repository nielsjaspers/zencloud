import React from "react";

export default function FileList({ files, loading, error, refreshFiles }) {
    // Direct download using a blob
    const handleDownload = async (file) => {
        try {
            const response = await fetch(
                `http://localhost:8080/download?id=${file.id}`
            );
            if (!response.ok) {
                throw new Error("Error downloading file");
            }
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = file.filename;
            document.body.appendChild(a);
            a.click();
            a.remove();
            window.URL.revokeObjectURL(url);
        } catch (err) {
            alert("Error downloading file: " + err.message);
        }
    };

    const handleDelete = async (file) => {
        try {
            const response = await fetch(
                `http://localhost:8080/delete?id=${file.id}`,
                {
                    method: "DELETE",
                }
            );
            if (!response.ok) {
                throw new Error("Error deleting file");
            }
            // Refresh file list after deletion
            if (refreshFiles) {
                refreshFiles();
            }
        } catch (err) {
            alert("Error deleting file: " + err.message);
        }
    };

    if (loading) return <div>Loading files...</div>;
    if (error) return <div>Error: {error}</div>;

    return (
        <div>
            <h3>Files on Server</h3>
            {files.length === 0 ? (
                <p>No files found.</p>
            ) : (
                <ul>
                    {files.map((file) => (
                        <li key={file.id}>
                            {file.filename}{" "}
                            <button onClick={() => handleDownload(file)}>Download</button>{" "}
                            <button onClick={() => handleDelete(file)}>Delete</button>
                        </li>
                    ))}
                </ul>
            )}
        </div>
    );
}
