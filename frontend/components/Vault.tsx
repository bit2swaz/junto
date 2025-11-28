"use client";

import { useState, useEffect } from "react";
import { useAuth } from "@/context/AuthContext";

interface VaultItem {
  id: number;
  content_text: string;
  unlock_at: string;
  locked: boolean;
  created_by: number;
}

export default function Vault() {
  const { token, user } = useAuth();
  const [items, setItems] = useState<VaultItem[]>([]);
  const [content, setContent] = useState("");
  const [unlockMinutes, setUnlockMinutes] = useState(5);
  const [isLoading, setIsLoading] = useState(false);

  const fetchItems = async () => {
    if (!token) return;
    try {
      const res = await fetch("http://localhost:8080/vault", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setItems(data || []);
      }
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    fetchItems();
    const interval = setInterval(fetchItems, 10000); // Poll every 10s
    return () => clearInterval(interval);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const addToVault = async () => {
    if (!content) return;
    setIsLoading(true);
    try {
      const unlockAt = new Date(Date.now() + unlockMinutes * 60000).toISOString();
      const res = await fetch("http://localhost:8080/vault", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ content, unlock_at: unlockAt }),
      });
      if (res.ok) {
        setContent("");
        fetchItems();
      }
    } catch (err) {
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="p-4 bg-white rounded-lg shadow max-w-md mx-auto mt-4">
      <h2 className="text-xl font-bold mb-4">Time Vault</h2>
      
      <div className="mb-6 space-y-2">
        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="Write a secret note..."
          className="w-full p-2 border rounded h-24"
        />
        <div className="flex gap-2 items-center">
          <span className="text-sm text-gray-600">Unlock in:</span>
          <select 
            value={unlockMinutes}
            onChange={(e) => setUnlockMinutes(Number(e.target.value))}
            className="p-1 border rounded"
          >
            <option value={1}>1 min</option>
            <option value={5}>5 mins</option>
            <option value={60}>1 hour</option>
            <option value={1440}>24 hours</option>
          </select>
          <button
            onClick={addToVault}
            disabled={isLoading || !content}
            className="ml-auto px-4 py-1 bg-purple-600 text-white rounded hover:bg-purple-700 disabled:opacity-50"
          >
            {isLoading ? "Locking..." : "Lock Away"}
          </button>
        </div>
      </div>

      <div className="space-y-3">
        {items.map((item) => (
          <div key={item.id} className={`p-3 rounded border ${item.locked ? 'bg-gray-100' : 'bg-yellow-50 border-yellow-200'}`}>
            <div className="flex justify-between items-start mb-1">
              <span className="text-xs text-gray-500">
                {item.created_by === user?.id ? "You" : "Partner"} • {new Date(item.unlock_at).toLocaleString()}
              </span>
              {item.locked && <span className="text-xs font-bold text-gray-500">🔒 LOCKED</span>}
            </div>
            
            {item.locked ? (
              <div className="text-gray-400 italic text-sm">
                {item.created_by === user?.id ? (
                  <>
                    <p className="not-italic text-gray-800">{item.content_text}</p>
                    <p className="text-xs mt-1">(Visible only to you until unlock)</p>
                  </>
                ) : (
                  "This content is locked until the timer expires."
                )}
              </div>
            ) : (
              <p className="text-gray-800">{item.content_text}</p>
            )}
          </div>
        ))}
        {items.length === 0 && <p className="text-center text-gray-400 text-sm">Vault is empty</p>}
      </div>
    </div>
  );
}
