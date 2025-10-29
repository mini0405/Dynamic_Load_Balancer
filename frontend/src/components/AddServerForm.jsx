import React, { useState } from 'react';

const AddServerForm = ({ onRefresh, className = '', title = 'Add New Server' }) => {
    const [address, setAddress] = useState('localhost');
    const [port, setPort] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        
        try {
            await fetch('/api/servers/add', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    address: address,
                    port: parseInt(port)
                }),
            });
            if (onRefresh) { onRefresh(); }
        } catch (error) {
            console.error('Error adding server:', error);
        }
    };

    return (
        <div className={`add-server ${className}`.trim()}>
            <h2>{title}</h2>
            <form onSubmit={handleSubmit}>
                <input
                    type="text"
                    value={address}
                    onChange={(e) => setAddress(e.target.value)}
                    placeholder="Address"
                />
                <input
                    type="number"
                    value={port}
                    onChange={(e) => setPort(e.target.value)}
                    placeholder="Port"
                />
                <button className="cyber-button" type="submit">Add Server</button>
            </form>
        </div>
    );
};

export default AddServerForm;
