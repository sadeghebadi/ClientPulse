"use client";

import React, { useEffect, useRef, useState, useCallback } from "react";
import * as THREE from "three";
import { useReducedMotion } from "./motion/useReducedMotion";
import { Activity, ShieldCheck, Globe, Zap } from "lucide-react";

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

interface SatelliteData {
  id: string;
  name: string;
  type: "api" | "web" | "worker" | "edge";
  latency: number;
  status: "up" | "verified";
  position: THREE.Vector3;
  mesh?: THREE.Mesh;
  glowMesh?: THREE.Mesh;
  curve?: THREE.CatmullRomCurve3;
  lineMesh?: THREE.Line;
  packetOffset: number;
  packetSpeed: number;
  packetMesh?: THREE.Mesh;
}

const INITIAL_SATELLITES: Omit<SatelliteData, "position" | "packetOffset" | "packetSpeed">[] = [
  { id: "api-gw", name: "api.gateway.internal", type: "api", latency: 22, status: "up" },
  { id: "web-store", name: "storefront.production", type: "web", latency: 18, status: "up" },
  { id: "auth-svc", name: "auth.sso.service", type: "api", latency: 26, status: "up" },
  { id: "db-rep", name: "db.primary.replica", type: "worker", latency: 12, status: "up" },
  { id: "queue-wrk", name: "queue.dispatcher.cron", type: "worker", latency: 9, status: "up" },
  { id: "cdn-edge", name: "cdn.edge.frankfurt", type: "edge", latency: 15, status: "up" },
  { id: "pay-hook", name: "webhook.stripe.rx", type: "api", latency: 31, status: "up" },
  { id: "cron-bk", name: "cron.nightly.db-backup", type: "worker", latency: 6, status: "up" },
  { id: "dns-res", name: "dns.edge.tehran", type: "edge", latency: 14, status: "up" },
  { id: "cache-rd", name: "redis.cache.cluster", type: "worker", latency: 7, status: "up" },
  { id: "edge-us", name: "probe.us-east.edge", type: "edge", latency: 28, status: "up" },
  { id: "edge-ap", name: "probe.singapore.edge", type: "edge", latency: 46, status: "up" },
];

export function HeroNetworkCanvas() {
  const mountRef = useRef<HTMLDivElement>(null);
  const [hasWebGL, setHasWebGL] = useState(true);
  const [selectedSatellite, setSelectedSatellite] = useState<SatelliteData | null>(null);
  const [hoveredSatellite, setHoveredSatellite] = useState<SatelliteData | null>(null);
  const reducedMotion = useReducedMotion();

  // Pointer position normalized to [-1, 1]
  const mouseRef = useRef({ x: 0, y: 0, targetX: 0, targetY: 0 });

  const handlePointerMove = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    const x = ((e.clientX - rect.left) / rect.width) * 2 - 1;
    const y = -(((e.clientY - rect.top) / rect.height) * 2 - 1);
    mouseRef.current.targetX = x;
    mouseRef.current.targetY = y;
  }, []);

  useEffect(() => {
    if (!isWebGLAvailable()) {
      setHasWebGL(false);
      return;
    }

    const container = mountRef.current;
    if (!container) return;

    let animId: number;
    let isVisible = true;

    const width = container.clientWidth || 1200;
    const height = container.clientHeight || 640;

    // 1. Scene, Camera & Renderer
    const scene = new THREE.Scene();
    scene.fog = new THREE.FogExp2(0x070b14, 0.025);

    const camera = new THREE.PerspectiveCamera(45, width / height, 0.1, 100);
    camera.position.set(0, 1.5, 22);

    const renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: true,
      powerPreference: "high-performance",
    });
    renderer.setSize(width, height);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    renderer.setClearColor(0x000000, 0);
    container.appendChild(renderer.domElement);

    // 2. Lighting
    const ambientLight = new THREE.AmbientLight(0x3b82f6, 0.6);
    scene.add(ambientLight);

    const blueLight = new THREE.PointLight(0x3b82f6, 2.5, 30);
    blueLight.position.set(-6, 4, 8);
    scene.add(blueLight);

    const cyanLight = new THREE.PointLight(0x06b6d4, 2.0, 30);
    cyanLight.position.set(6, -4, 8);
    scene.add(cyanLight);

    const violetLight = new THREE.PointLight(0x8b5cf6, 1.8, 25);
    violetLight.position.set(0, 6, -6);
    scene.add(violetLight);

    // Group for everything that rotates slowly
    const universeGroup = new THREE.Group();
    scene.add(universeGroup);

    // 3. Central Core: ClientPulse High-Throughput Verification Engine
    const coreGroup = new THREE.Group();
    universeGroup.add(coreGroup);

    // Outer Geodesic Wireframe
    const outerGeo = new THREE.IcosahedronGeometry(2.4, 2);
    const outerMat = new THREE.MeshBasicMaterial({
      color: 0x3b82f6,
      wireframe: true,
      transparent: true,
      opacity: 0.35,
    });
    const outerMesh = new THREE.Mesh(outerGeo, outerMat);
    coreGroup.add(outerMesh);

    // Middle Crystalline Polyhedron
    const midGeo = new THREE.OctahedronGeometry(1.6, 1);
    const midMat = new THREE.MeshBasicMaterial({
      color: 0x06b6d4,
      wireframe: true,
      transparent: true,
      opacity: 0.55,
    });
    const midMesh = new THREE.Mesh(midGeo, midMat);
    coreGroup.add(midMesh);

    // Inner Luminous Core
    const innerGeo = new THREE.SphereGeometry(0.9, 24, 24);
    const innerMat = new THREE.MeshBasicMaterial({
      color: 0x60a5fa,
      transparent: true,
      opacity: 0.85,
    });
    const innerMesh = new THREE.Mesh(innerGeo, innerMat);
    coreGroup.add(innerMesh);

    // Triple Orbital Rings
    const ringMat1 = new THREE.MeshBasicMaterial({ color: 0x818cf8, transparent: true, opacity: 0.4 });
    const ringMat2 = new THREE.MeshBasicMaterial({ color: 0x38bdf8, transparent: true, opacity: 0.35 });
    const ringMat3 = new THREE.MeshBasicMaterial({ color: 0x34d399, transparent: true, opacity: 0.3 });

    const ring1 = new THREE.Mesh(new THREE.TorusGeometry(3.8, 0.025, 16, 80), ringMat1);
    ring1.rotation.x = Math.PI / 3;
    ring1.rotation.y = Math.PI / 8;
    coreGroup.add(ring1);

    const ring2 = new THREE.Mesh(new THREE.TorusGeometry(4.4, 0.025, 16, 80), ringMat2);
    ring2.rotation.x = -Math.PI / 4;
    ring2.rotation.z = Math.PI / 6;
    coreGroup.add(ring2);

    const ring3 = new THREE.Mesh(new THREE.TorusGeometry(4.9, 0.02, 16, 80), ringMat3);
    ring3.rotation.y = Math.PI / 3;
    coreGroup.add(ring3);

    // 4. Starfield Ambient Particles
    const starCount = 300;
    const starGeo = new THREE.BufferGeometry();
    const starPositions = new Float32Array(starCount * 3);
    for (let i = 0; i < starCount * 3; i += 3) {
      starPositions[i] = (Math.random() - 0.5) * 45;
      starPositions[i + 1] = (Math.random() - 0.5) * 30;
      starPositions[i + 2] = (Math.random() - 0.5) * 35;
    }
    starGeo.setAttribute("position", new THREE.BufferAttribute(starPositions, 3));
    const starMat = new THREE.PointsMaterial({
      color: 0x64748b,
      size: 0.07,
      transparent: true,
      opacity: 0.45,
    });
    const starField = new THREE.Points(starGeo, starMat);
    scene.add(starField);

    // 5. Build 12 Distributed Satellite Nodes with Curved Splines & Light Packets
    const satellitesGroup = new THREE.Group();
    universeGroup.add(satellitesGroup);

    const satellites: SatelliteData[] = [];
    const raycastMeshes: THREE.Mesh[] = [];

    INITIAL_SATELLITES.forEach((item, idx) => {
      const angle = (idx / INITIAL_SATELLITES.length) * Math.PI * 2;
      const radius = 6.8 + (idx % 4) * 1.3;
      const y = (idx % 2 === 0 ? 1 : -1) * (1.4 + (idx % 3) * 1.1);
      const z = Math.sin(angle * 1.5) * 3.5;

      const pos = new THREE.Vector3(
        Math.cos(angle) * radius,
        y,
        Math.sin(angle) * radius + z
      );

      // Node color by type
      let nodeColor = 0x10b981; // emerald
      if (item.type === "api") nodeColor = 0x38bdf8; // sky
      if (item.type === "worker") nodeColor = 0xa855f7; // purple
      if (item.type === "edge") nodeColor = 0x06b6d4; // cyan

      // Satellite Sphere Mesh
      const nodeGeo = new THREE.SphereGeometry(0.32, 16, 16);
      const nodeMat = new THREE.MeshBasicMaterial({
        color: nodeColor,
        transparent: true,
        opacity: 0.95,
      });
      const nodeMesh = new THREE.Mesh(nodeGeo, nodeMat);
      nodeMesh.position.copy(pos);
      nodeMesh.userData = { id: item.id };
      satellitesGroup.add(nodeMesh);
      raycastMeshes.push(nodeMesh);

      // Outer Halo Ring
      const haloGeo = new THREE.RingGeometry(0.42, 0.48, 24);
      const haloMat = new THREE.MeshBasicMaterial({
        color: nodeColor,
        transparent: true,
        opacity: 0.4,
        side: THREE.DoubleSide,
      });
      const haloMesh = new THREE.Mesh(haloGeo, haloMat);
      haloMesh.position.copy(pos);
      satellitesGroup.add(haloMesh);

      // CatmullRom Curved Connection to Core (0, 0, 0)
      const midPoint = new THREE.Vector3(
        pos.x * 0.5 + (idx % 2 === 0 ? 0.8 : -0.8),
        pos.y * 0.5 + 0.6,
        pos.z * 0.5
      );
      const curve = new THREE.CatmullRomCurve3([
        new THREE.Vector3(0, 0, 0),
        midPoint,
        pos,
      ]);

      const curvePoints = curve.getPoints(36);
      const lineGeo = new THREE.BufferGeometry().setFromPoints(curvePoints);
      const lineMat = new THREE.LineBasicMaterial({
        color: nodeColor,
        transparent: true,
        opacity: 0.22,
      });
      const lineMesh = new THREE.Line(lineGeo, lineMat);
      satellitesGroup.add(lineMesh);

      // Travelling Light Packet Mesh
      const packetGeo = new THREE.SphereGeometry(0.12, 10, 10);
      const packetMat = new THREE.MeshBasicMaterial({
        color: 0xffffff,
        transparent: true,
        opacity: 0.95,
      });
      const packetMesh = new THREE.Mesh(packetGeo, packetMat);
      satellitesGroup.add(packetMesh);

      satellites.push({
        ...item,
        position: pos,
        mesh: nodeMesh,
        glowMesh: haloMesh,
        curve,
        lineMesh,
        packetOffset: (idx / INITIAL_SATELLITES.length),
        packetSpeed: 0.004 + (idx % 3) * 0.0015,
        packetMesh,
      });
    });

    // Default selection
    setSelectedSatellite(satellites[0]);

    // 6. Raycasting Setup for Interactivity
    const raycaster = new THREE.Raycaster();
    const mouseVector = new THREE.Vector2();

    const handleContainerClick = (e: MouseEvent) => {
      const rect = container.getBoundingClientRect();
      mouseVector.x = ((e.clientX - rect.left) / rect.width) * 2 - 1;
      mouseVector.y = -(((e.clientY - rect.top) / rect.height) * 2 - 1);
      raycaster.setFromCamera(mouseVector, camera);

      const intersects = raycaster.intersectObjects(raycastMeshes);
      if (intersects.length > 0) {
        const clickedMesh = intersects[0].object as THREE.Mesh;
        const matched = satellites.find((s) => s.id === clickedMesh.userData.id);
        if (matched) {
          setSelectedSatellite(matched);
          // Quick pulse scaling animation on click
          clickedMesh.scale.set(1.6, 1.6, 1.6);
        }
      }
    };

    const handlePointerHover = (e: MouseEvent) => {
      const rect = container.getBoundingClientRect();
      mouseVector.x = ((e.clientX - rect.left) / rect.width) * 2 - 1;
      mouseVector.y = -(((e.clientY - rect.top) / rect.height) * 2 - 1);
      raycaster.setFromCamera(mouseVector, camera);

      const intersects = raycaster.intersectObjects(raycastMeshes);
      if (intersects.length > 0) {
        const hoveredMesh = intersects[0].object as THREE.Mesh;
        const matched = satellites.find((s) => s.id === hoveredMesh.userData.id);
        setHoveredSatellite(matched || null);
        container.style.cursor = "pointer";
      } else {
        setHoveredSatellite(null);
        container.style.cursor = "default";
      }
    };

    container.addEventListener("click", handleContainerClick);
    container.addEventListener("pointermove", handlePointerHover);

    // 7. Responsive Resizing
    const handleResize = () => {
      if (!container) return;
      const w = container.clientWidth || 1200;
      const h = container.clientHeight || 640;
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
      renderer.setSize(w, h);
    };
    window.addEventListener("resize", handleResize);

    // 8. Intersection Observer to Pause RAF when Off-screen
    const observer = new IntersectionObserver(
      (entries) => {
        isVisible = entries[0].isIntersecting;
      },
      { threshold: 0.05 }
    );
    observer.observe(container);

    // 9. Animation Loop (60fps lerped)
    let clock = new THREE.Clock();

    const animate = () => {
      animId = requestAnimationFrame(animate);
      if (!isVisible) return;

      const delta = clock.getDelta();
      const time = clock.getElapsedTime();

      // Smooth pointer parallax lerping
      mouseRef.current.x += (mouseRef.current.targetX - mouseRef.current.x) * 0.05;
      mouseRef.current.y += (mouseRef.current.targetY - mouseRef.current.y) * 0.05;

      camera.position.x = mouseRef.current.x * 2.5;
      camera.position.y = 1.5 + mouseRef.current.y * 1.5;
      camera.lookAt(0, 0, 0);

      // Core rotation & breathing pulse
      if (!reducedMotion) {
        outerMesh.rotation.y += 0.004;
        outerMesh.rotation.x += 0.002;
        midMesh.rotation.y -= 0.007;
        midMesh.rotation.z += 0.003;

        const pulseScale = 1 + Math.sin(time * 3) * 0.06;
        innerMesh.scale.set(pulseScale, pulseScale, pulseScale);

        ring1.rotation.z += 0.005;
        ring2.rotation.y += 0.006;
        ring3.rotation.x += 0.004;

        universeGroup.rotation.y = time * 0.035;

        // Animate Data Packets along Splines
        satellites.forEach((sat) => {
          sat.packetOffset = (sat.packetOffset + sat.packetSpeed) % 1;
          if (sat.curve && sat.packetMesh) {
            const p = sat.curve.getPoint(sat.packetOffset);
            sat.packetMesh.position.copy(p);
          }

          // Return clicked mesh scale smoothly back to 1
          if (sat.mesh && sat.mesh.scale.x > 1) {
            sat.mesh.scale.subScalar(delta * 2);
            if (sat.mesh.scale.x < 1) sat.mesh.scale.set(1, 1, 1);
          }

          // Orient halo rings towards camera
          if (sat.glowMesh) {
            sat.glowMesh.lookAt(camera.position);
          }
        });
      }

      renderer.render(scene, camera);
    };

    animate();

    // 10. Strict Cleanup & GPU Disposal
    return () => {
      cancelAnimationFrame(animId);
      observer.disconnect();
      window.removeEventListener("resize", handleResize);
      container.removeEventListener("click", handleContainerClick);
      container.removeEventListener("pointermove", handlePointerHover);

      // Geometries & Materials Disposal
      outerGeo.dispose();
      outerMat.dispose();
      midGeo.dispose();
      midMat.dispose();
      innerGeo.dispose();
      innerMat.dispose();
      ringMat1.dispose();
      ringMat2.dispose();
      ringMat3.dispose();
      starGeo.dispose();
      starMat.dispose();

      satellites.forEach((s) => {
        s.mesh?.geometry.dispose();
        (s.mesh?.material as THREE.Material)?.dispose();
        s.glowMesh?.geometry.dispose();
        (s.glowMesh?.material as THREE.Material)?.dispose();
        s.lineMesh?.geometry.dispose();
        (s.lineMesh?.material as THREE.Material)?.dispose();
        s.packetMesh?.geometry.dispose();
        (s.packetMesh?.material as THREE.Material)?.dispose();
      });

      renderer.dispose();
      if (renderer.domElement && container.contains(renderer.domElement)) {
        container.removeChild(renderer.domElement);
      }
    };
  }, [reducedMotion]);

  if (!hasWebGL) {
    return (
      <div className="w-full h-[480px] sm:h-[580px] lg:h-[680px] flex items-center justify-center relative overflow-hidden bg-gradient-to-b from-blue-950/20 to-transparent rounded-3xl border border-blue-900/30">
        <div className="relative text-center p-6 space-y-4 max-w-md">
          <div className="w-16 h-16 mx-auto rounded-2xl bg-blue-600/20 border border-blue-500/40 text-blue-400 flex items-center justify-center">
            <Globe className="w-8 h-8 animate-pulse" />
          </div>
          <h3 className="text-lg font-bold text-white">Live 3D Network Active</h3>
          <p className="text-xs text-slate-400">
            Distributed health verification probes continuously querying global edge nodes.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="relative w-full h-[520px] sm:h-[620px] lg:h-[720px] overflow-hidden select-none">
      {/* 3D WebGL Canvas Mount Container */}
      <div
        ref={mountRef}
        onPointerMove={handlePointerMove}
        className="w-full h-full cursor-grab active:cursor-grabbing"
      />

      {/* Floating Spatial HUD: Active Node Telemetry Card */}
      {(hoveredSatellite || selectedSatellite) && (
        <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-20 pointer-events-none transition-all duration-300">
          <div className="flex items-center gap-3 px-4 py-2.5 rounded-2xl bg-[#090e1a]/90 border border-blue-500/30 shadow-2xl backdrop-blur-xl ring-1 ring-white/10 text-xs">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse shadow-[0_0_8px_#34d399]" />
            <div className="flex items-center gap-2">
              <span className="font-mono font-bold text-white tracking-wide">
                {(hoveredSatellite || selectedSatellite)?.name}
              </span>
              <span className="px-2 py-0.5 rounded-md text-[10px] font-semibold uppercase tracking-wider bg-blue-500/20 text-blue-300 border border-blue-500/30">
                {(hoveredSatellite || selectedSatellite)?.type}
              </span>
            </div>
            <div className="h-3 w-px bg-slate-700 mx-1" />
            <div className="flex items-center gap-1.5 text-emerald-400 font-mono font-bold">
              <Zap className="w-3.5 h-3.5" />
              <span>{(hoveredSatellite || selectedSatellite)?.latency} ms</span>
            </div>
          </div>
        </div>
      )}

      {/* Subtle Hint Pill */}
      <div className="absolute top-4 right-4 z-10 pointer-events-none hidden sm:flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-900/60 border border-slate-800 text-[11px] text-slate-400 backdrop-blur-md">
        <Activity className="w-3 h-3 text-blue-400" />
        <span>Click node to inspect telemetry</span>
      </div>
    </div>
  );
}
