"use client";

import { useEffect, useRef, useState, useMemo } from 'react';
import { useWebSocket } from '@/context/WebSocketContext';
import throttle from 'lodash/throttle';

export default function RoomCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const playerPos = useRef({ x: 150, y: 150 });
  const partnerPos = useRef<{ x: number; y: number } | null>(null);
  const displayedPartnerPos = useRef<{ x: number; y: number } | null>(null);
  
  const { sendMessage, isConnected, subscribe } = useWebSocket();
  
  // Joystick state
  const joystickRef = useRef<HTMLDivElement>(null);
  const [joystickPos, setJoystickPos] = useState({ x: 0, y: 0 });
  const isDragging = useRef(false);
  const movementVector = useRef({ x: 0, y: 0 });
  const speed = 2;

  // Touch/Haptic state
  const [isPartnerTouching, setIsPartnerTouching] = useState(false);

  // Throttle sending updates to 30 times per second (~33ms)
  const sendMoveUpdate = useMemo(
    () =>
      throttle((x: number, y: number) => {
        sendMessage({ type: 'move', x, y });
      }, 33),
    [sendMessage]
  );

  // Subscribe to messages
  useEffect(() => {
    const unsubscribe = subscribe((msg) => {
      if (msg.type === 'move') {
        partnerPos.current = { x: msg.x, y: msg.y };
        if (!displayedPartnerPos.current) {
          displayedPartnerPos.current = { x: msg.x, y: msg.y };
        }
      } else if (msg.type === 'TOUCH_START') {
        setIsPartnerTouching(true);
      } else if (msg.type === 'TOUCH_END') {
        setIsPartnerTouching(false);
      }
    });
    return unsubscribe;
  }, [subscribe]);

  // Haptic feedback loop
  useEffect(() => {
    let interval: NodeJS.Timeout;
    if (isPartnerTouching) {
      const vibrate = () => {
        if (typeof navigator !== 'undefined' && navigator.vibrate) {
          navigator.vibrate([100, 50]);
        }
      };
      // Initial trigger
      vibrate();
      // Loop
      interval = setInterval(vibrate, 150);
    }
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [isPartnerTouching]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animationFrameId: number;

    const lerp = (start: number, end: number, t: number) => {
      return start * (1 - t) + end * t;
    };

    const render = () => {
      // Update position based on joystick vector
      if (movementVector.current.x !== 0 || movementVector.current.y !== 0) {
        playerPos.current.x += movementVector.current.x * speed;
        playerPos.current.y += movementVector.current.y * speed;

        // Boundary checks
        playerPos.current.x = Math.max(0, Math.min(canvas.width - 20, playerPos.current.x));
        playerPos.current.y = Math.max(0, Math.min(canvas.height - 20, playerPos.current.y));

        // Send update to server
        if (isConnected) {
          sendMoveUpdate(playerPos.current.x, playerPos.current.y);
        }
      }

      // Clear canvas
      ctx.clearRect(0, 0, canvas.width, canvas.height);

      // Draw partner (blue square) with Lerp
      if (partnerPos.current && displayedPartnerPos.current) {
        displayedPartnerPos.current.x = lerp(displayedPartnerPos.current.x, partnerPos.current.x, 0.1);
        displayedPartnerPos.current.y = lerp(displayedPartnerPos.current.y, partnerPos.current.y, 0.1);
        
        ctx.fillStyle = 'blue';
        ctx.fillRect(displayedPartnerPos.current.x, displayedPartnerPos.current.y, 20, 20);
      }

      // Draw player (red square)
      ctx.fillStyle = 'red';
      ctx.fillRect(playerPos.current.x, playerPos.current.y, 20, 20);

      animationFrameId = requestAnimationFrame(render);
    };

    render();

    return () => {
      cancelAnimationFrame(animationFrameId);
      sendMoveUpdate.cancel();
    };
  }, [isConnected, sendMoveUpdate]);

  const handleCanvasPointerDown = () => {
    if (isConnected) {
      sendMessage({ type: 'TOUCH_START' });
    }
  };

  const handleCanvasPointerUp = () => {
    if (isConnected) {
      sendMessage({ type: 'TOUCH_END' });
    }
  };

  const handleTouchStart = (e: React.TouchEvent | React.MouseEvent) => {
    isDragging.current = true;
    updateJoystick(e);
  };

  const handleTouchMove = (e: React.TouchEvent | React.MouseEvent) => {
    if (!isDragging.current) return;
    updateJoystick(e);
  };

  const handleTouchEnd = () => {
    isDragging.current = false;
    setJoystickPos({ x: 0, y: 0 });
    movementVector.current = { x: 0, y: 0 };
  };

  const updateJoystick = (e: React.TouchEvent | React.MouseEvent) => {
    const joystick = joystickRef.current;
    if (!joystick) return;

    const rect = joystick.parentElement?.getBoundingClientRect();
    if (!rect) return;

    let clientX, clientY;
    if ('touches' in e) {
      clientX = e.touches[0].clientX;
      clientY = e.touches[0].clientY;
    } else {
      clientX = (e as React.MouseEvent).clientX;
      clientY = (e as React.MouseEvent).clientY;
    }

    const centerX = rect.width / 2;
    const centerY = rect.height / 2;

    // Calculate distance from center
    let dx = clientX - rect.left - centerX;
    let dy = clientY - rect.top - centerY;

    // Clamp joystick movement to radius
    const radius = 40;
    const distance = Math.sqrt(dx * dx + dy * dy);
    
    if (distance > radius) {
      const angle = Math.atan2(dy, dx);
      dx = Math.cos(angle) * radius;
      dy = Math.sin(angle) * radius;
    }

    setJoystickPos({ x: dx, y: dy });

    // Normalize vector for movement
    if (distance > 0) {
      // Normalize to 0-1 range based on max radius
      const normalizedDistance = Math.min(distance, radius) / radius;
      const angle = Math.atan2(dy, dx);
      movementVector.current = {
        x: Math.cos(angle) * normalizedDistance,
        y: Math.sin(angle) * normalizedDistance
      };
    } else {
      movementVector.current = { x: 0, y: 0 };
    }
  };

  return (
    <div className="relative w-full max-w-[300px] mx-auto">
      {isPartnerTouching && (
        <div className="fixed inset-0 z-50 pointer-events-none bg-gradient-to-r from-pink-500/30 to-orange-500/30 animate-pulse" />
      )}
      <canvas
        ref={canvasRef}
        width={300}
        height={300}
        className="border border-gray-300 rounded-lg bg-white w-full touch-none"
        onPointerDown={handleCanvasPointerDown}
        onPointerUp={handleCanvasPointerUp}
        onPointerLeave={handleCanvasPointerUp}
      />
      
      {/* Virtual Joystick Overlay */}
      <div 
        className="absolute bottom-4 right-4 w-24 h-24 bg-gray-200/50 rounded-full touch-none flex items-center justify-center"
        onMouseDown={handleTouchStart}
        onMouseMove={handleTouchMove}
        onMouseUp={handleTouchEnd}
        onMouseLeave={handleTouchEnd}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        <div 
          ref={joystickRef}
          className="w-10 h-10 bg-blue-500/80 rounded-full shadow-lg pointer-events-none"
          style={{
            transform: `translate(${joystickPos.x}px, ${joystickPos.y}px)`,
            transition: isDragging.current ? 'none' : 'transform 0.1s ease-out'
          }}
        />
      </div>
    </div>
  );
}

