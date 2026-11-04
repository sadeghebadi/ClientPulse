"use client";

import React, { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import { useReducedMotion } from "./motion/useReducedMotion";

function isWebGLAvailable(): boolean {
  try {
    const canvas = document.createElement("canvas");
    return !!(
      window.WebGLRenderingContext &&
      (canvas.getContext("webgl") || canvas.getContext("experimental-webgl"))
    );
  } catch {
    return false;
  }
}

interface NodeData {
  pos: THREE.Vector3;
  targetPos: THREE.Vector3;
  color: THREE.Color;
  label: string;
  size: number;
}

export function HeroNetworkCanvas() {
  const mountRef = useRef<HTMLDivElement>(null);
  const [hasWebGL, setHasWebGL] = useState(true);
  const reducedMotion = useReducedMotion();

  useEffect(() => {
    if (!isWebGLAvailable()) {
      setHasWebGL(false);
      return;
    }

    const container = mountRef.current;
    if (!container) return;

    // Scene & Camera Setup
    const width = container.clientWidth || 800;
    const height = container.clientHeight || 500;

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(45, width / height, 0.1, 1000);
    camera.position.set(0, 0, 18);

    const renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: true,
      powerPreference: "high-performance",
    });
    renderer.setSize(width, height);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    renderer.setClearColor(0x000000, 0);
    container.appendChild(renderer.domElement);

    // Group for all rotating objects
    const networkGroup = new THREE.Group();
    scene.add(networkGroup);

    // 1. Central Core Node (ClientPulse Engine)
    const coreGeo = new THREE.IcosahedronGeometry(2.2, 2);
    const coreMat = new THREE.MeshBasicMaterial({
      color: 0x3b82f6,
      wireframe: true,
      transparent: true,
      opacity: 0.35,
    });
    const coreMesh = new THREE.Mesh(coreGeo, coreMat);
    networkGroup.add(coreMesh);

    // Core inner sphere
    const innerGeo = new THREE.SphereGeometry(1.2, 16, 16);
    const innerMat = new THREE.MeshBasicMaterial({
      color: 0x60a5fa,
      wireframe: false,
      transparent: true,
      opacity: 0.85,
    });
    const innerMesh = new THREE.Mesh(innerGeo, innerMat);
    networkGroup.add(innerMesh);

    // Outer orbital ring
    const ringGeo = new THREE.TorusGeometry(3.6, 0.03, 16, 64);
    const ringMat = new THREE.MeshBasicMaterial({
      color: 0x818cf8,
      transparent: true,
      opacity: 0.4,
    });
    const ringMesh = new THREE.Mesh(ringGeo, ringMat);
    ringMesh.rotation.x = Math.PI / 3;
    networkGroup.add(ringMesh);

    // Second tilted ring
    const ring2Mesh = ringMesh.clone();
    ring2Mesh.rotation.x = -Math.PI / 4;
    ring2Mesh.rotation.y = Math.PI / 6;
    networkGroup.add(ring2Mesh);

    // 2. Distributed Satellite Nodes (Monitored Targets & Probes)
    const satelliteCount = 8;
    const satellites: { mesh: THREE.Mesh; angle: number; speed: number; radius: number; y: number }[] = [];
    const linesGroup = new THREE.Group();
    networkGroup.add(linesGroup);

    const nodeGeo = new THREE.SphereGeometry(0.35, 12, 12);
    const colors = [0x10b981, 0x06b6d4, 0x3b82f6, 0x8b5cf6, 0x10b981, 0x3b82f6, 0x06b6d4, 0x10b981];

    for (let i = 0; i < satelliteCount; i++) {
      const angle = (i / satelliteCount) * Math.PI * 2;
      const radius = 5.5 + (i % 3) * 1.2;
      const y = (i % 2 === 0 ? 1 : -1) * (1.2 + (i % 3) * 0.8);

      const mat = new THREE.MeshBasicMaterial({
        color: colors[i],
        transparent: true,
        opacity: 0.9,
      });
      const nodeMesh = new THREE.Mesh(nodeGeo, mat);
      nodeMesh.position.set(Math.cos(angle) * radius, y, Math.sin(angle) * radius);
      networkGroup.add(nodeMesh);

      satellites.push({
        mesh: nodeMesh,
        angle,
        speed: 0.003 + (i % 3) * 0.001,
        radius,
        y,
      });
    }

    // 3. Dynamic Lines Connecting Core to Satellites
    const lineMat = new THREE.LineBasicMaterial({
      color: 0x3b82f6,
      transparent: true,
      opacity: 0.25,
    });

    const lineObjects: THREE.Line[] = [];
    for (let i = 0; i < satelliteCount; i++) {
      const points = [new THREE.Vector3(0, 0, 0), satellites[i].mesh.position];
      const lineGeometry = new THREE.BufferGeometry().setFromPoints(points);
      const line = new THREE.Line(lineGeometry, lineMat);
      linesGroup.add(line);
      lineObjects.push(line);
    }

    // 4. Glowing Data Stream Pulse Particles Traveling Along Lines
    const pulseCount = 16;
    const pulseGeo = new THREE.SphereGeometry(0.12, 8, 8);
    const pulseMat = new THREE.MeshBasicMaterial({
      color: 0x38bdf8,
      transparent: true,
      opacity: 0.95,
    });

    const pulses: { mesh: THREE.Mesh; satelliteIndex: number; progress: number; speed: number }[] = [];
    for (let i = 0; i < pulseCount; i++) {
      const pulseMesh = new THREE.Mesh(pulseGeo, pulseMat);
      networkGroup.add(pulseMesh);
      pulses.push({
        mesh: pulseMesh,
        satelliteIndex: i % satelliteCount,
        progress: Math.random(),
        speed: 0.006 + Math.random() * 0.008,
      });
    }

    // Pointer Parallax
    let targetRotationX = 0;
    let targetRotationY = 0;
    let mouseX = 0;
    let mouseY = 0;

    const handlePointerMove = (e: MouseEvent) => {
      const rect = container.getBoundingClientRect();
      const x = (e.clientX - rect.left) / rect.width - 0.5;
      const y = (e.clientY - rect.top) / rect.height - 0.5;
      mouseX = x * 2;
      mouseY = y * 2;
    };

    window.addEventListener("mousemove", handlePointerMove, { passive: true });

    // Handle Window Resize
    const handleResize = () => {
      if (!container) return;
      const w = container.clientWidth;
      const h = container.clientHeight;
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
      renderer.setSize(w, h);
    };

    window.addEventListener("resize", handleResize);

    // Animation & Viewport Intersection Optimization
    let isVisible = true;
    const observer = new IntersectionObserver(
      ([entry]) => {
        isVisible = entry.isIntersecting;
      },
      { threshold: 0.05 }
    );
    observer.observe(container);

    let animationId: number;

    const animate = () => {
      animationId = requestAnimationFrame(animate);

      if (!isVisible) return;

      // Rotate group smoothly
      if (!reducedMotion) {
        targetRotationY = mouseX * 0.45;
        targetRotationX = -mouseY * 0.3;

        networkGroup.rotation.y += (targetRotationY - networkGroup.rotation.y) * 0.05 + 0.002;
        networkGroup.rotation.x += (targetRotationX - networkGroup.rotation.x) * 0.05;

        // Core gentle breathing
        const time = Date.now() * 0.0015;
        const scale = 1 + Math.sin(time) * 0.06;
        innerMesh.scale.set(scale, scale, scale);

        // Update satellites & lines
        satellites.forEach((sat, idx) => {
          sat.angle += sat.speed;
          sat.mesh.position.x = Math.cos(sat.angle) * sat.radius;
          sat.mesh.position.z = Math.sin(sat.angle) * sat.radius;

          // Update connecting line geometry
          const line = lineObjects[idx];
          if (line) {
            const positions = (line.geometry.attributes.position as THREE.BufferAttribute).array as Float32Array;
            positions[3] = sat.mesh.position.x;
            positions[4] = sat.mesh.position.y;
            positions[5] = sat.mesh.position.z;
            line.geometry.attributes.position.needsUpdate = true;
          }
        });

        // Update traveling data pulses
        pulses.forEach((pulse) => {
          pulse.progress += pulse.speed;
          if (pulse.progress > 1) {
            pulse.progress = 0;
            pulse.satelliteIndex = (pulse.satelliteIndex + 1) % satelliteCount;
          }
          const satPos = satellites[pulse.satelliteIndex].mesh.position;
          pulse.mesh.position.lerpVectors(new THREE.Vector3(0, 0, 0), satPos, pulse.progress);
        });
      }

      renderer.render(scene, camera);
    };

    animate();

    // Cleanup resources to prevent memory leaks
    return () => {
      cancelAnimationFrame(animationId);
      observer.disconnect();
      window.removeEventListener("mousemove", handlePointerMove);
      window.removeEventListener("resize", handleResize);

      // Dispose three.js objects
      coreGeo.dispose();
      coreMat.dispose();
      innerGeo.dispose();
      innerMat.dispose();
      ringGeo.dispose();
      ringMat.dispose();
      nodeGeo.dispose();
      pulseGeo.dispose();
      pulseMat.dispose();
      lineMat.dispose();

      satellites.forEach((s) => {
        (s.mesh.material as THREE.Material).dispose();
      });

      lineObjects.forEach((line) => {
        line.geometry.dispose();
      });

      renderer.dispose();
      if (renderer.domElement && container.contains(renderer.domElement)) {
        container.removeChild(renderer.domElement);
      }
    };
  }, [reducedMotion]);

  if (!hasWebGL) {
    // Elegant fallback when WebGL is unavailable
    return (
      <div className="w-full h-full flex items-center justify-center p-8">
        <div className="relative w-72 h-72 rounded-full border border-blue-500/20 bg-blue-950/20 flex items-center justify-center animate-pulse-slow">
          <div className="w-48 h-48 rounded-full border border-violet-500/30 flex items-center justify-center">
            <div className="w-24 h-24 rounded-full bg-blue-500/20 blur-md" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div
      ref={mountRef}
      className="w-full h-[360px] sm:h-[460px] lg:h-[540px] pointer-events-auto cursor-grab active:cursor-grabbing"
      aria-hidden="true"
    />
  );
}

