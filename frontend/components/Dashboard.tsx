"use client";

import { useState } from "react";
import { useAuth } from "@/context/AuthContext";
import RoomCanvas from "./RoomCanvas";
import Vault from "./Vault";

export default function Dashboard() {
  const { user, token, refreshUser } = useAuth();
  const [pairingCode, setPairingCode] = useState("");
  const [partnerCode, setPartnerCode] = useState("");
  const [error, setError] = useState("");

  const generateCode = async () => {
    try {
      const res = await fetch("http://localhost:8080/couples/code", {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      setPairingCode(data.code);
    } catch {
      setError("Failed to generate code");
    }
  };

  const linkPartner = async () => {
    try {
      const res = await fetch("http://localhost:8080/couples/link", {
        method: "POST",
        headers: { 
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}` 
        },
        body: JSON.stringify({ code: partnerCode }),
      });
      if (!res.ok) throw new Error("Failed to link");
      
      await refreshUser(); // Refresh to get couple_id
    } catch {
      setError("Failed to link partner");
    }
  };

  if (!user) return <div>Loading...</div>;

  if (user.couple_id) {
    return (
      <div className="h-[calc(100vh-100px)] flex flex-col md:flex-row gap-4 p-4">
        <div className="flex-1 flex items-center justify-center bg-gray-50 rounded-lg">
          <RoomCanvas />
        </div>
        <div className="w-full md:w-96">
          <Vault />
        </div>
      </div>
    );
  }

  return (
    <div className="p-4 space-y-6">
      <h1 className="text-2xl font-bold">Connect with your Partner</h1>
      
      <div className="bg-white p-6 rounded-lg shadow">
        <h2 className="text-lg font-semibold mb-4">Your Pairing Code</h2>
        {pairingCode ? (
          <div className="text-3xl font-mono text-center py-4 bg-gray-100 rounded">
            {pairingCode}
          </div>
        ) : (
          <button
            onClick={generateCode}
            className="w-full py-2 px-4 bg-indigo-600 text-white rounded hover:bg-indigo-700"
          >
            Generate Code
          </button>
        )}
      </div>

      <div className="bg-white p-6 rounded-lg shadow">
        <h2 className="text-lg font-semibold mb-4">Enter Partner&apos;s Code</h2>
        <div className="flex gap-2">
          <input
            type="text"
            value={partnerCode}
            onChange={(e) => setPartnerCode(e.target.value)}
            className="flex-1 p-2 border rounded"
            placeholder="Enter 6-digit code"
          />
          <button
            onClick={linkPartner}
            className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
          >
            Link Partner
          </button>
        </div>
        {error && <p className="text-red-500 mt-2">{error}</p>}
      </div>
    </div>
  );
}
