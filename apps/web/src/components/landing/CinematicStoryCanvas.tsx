"use client";

import React, { useEffect, useRef, useCallback } from "react";
import * as THREE from "three";

interface CinematicStoryCanvasProps {
  progress: number; // 0.0 to 1.0 (scrubbed by GSAP ScrollTrigger)
  activeScene: number; // 0 to 5
  isRTL?: boolean;
}

interface NodeDef {
  id: string;
  name: string;
  type: "core" | "target" | "api" | "worker" | "edge";
  pos: [number, number, number];
  isTarget?: boolean;
}

const NODES_DATA: NodeDef[] = [
  // Primary target node that degrades and suffers total outage
  { id: "storefront", name: "storefront.production", type: "target", pos: [4.2, 0.6, -1.2], isTarget: true },

  // Edge & API nodes surrounding the space
  { id: "api-gw", name: "api.gateway.primary", type: "api", pos: [-3.8, 2.2, -0.5] },
  { id: "auth-svc", name: "auth.identity.service", type: "api", pos: [-2.4, -2.0, 1.8] },
  { id: "cdn-fra", name: "cdn.edge.frankfurt", type: "edge", pos: [2.8, 3.2, 2.2] },
  { id: "cdn-teh", name: "cdn.edge.tehran", type: "edge", pos: [5.6, -1.8, 1.5] },
  { id: "db-rep", name: "db.primary.replica", type: "worker", pos: [-5.2, -0.8, -2.5] },
  { id: "queue-wrk", name: "queue.dispatcher.cron", type: "worker", pos: [-1.5, 3.4, -3.2] },
  { id: "cache-rd", name: "redis.cache.cluster", type: "worker", pos: [1.8, -3.0, -1.8] },
  { id: "dns-edge", name: "dns.edge.global", type: "edge", pos: [6.0, 2.0, -3.0] },
  { id: "pay-hook", name: "webhook.stripe.rx", type: "api", pos: [3.4, -2.4, 3.2] },
  { id: "probe-us", name: "probe.us-east.edge", type: "edge", pos: [-6.2, 1.4, 2.5] },
  { id: "probe-sg", name: "probe.singapore.edge", type: "edge", pos: [-3.2, -3.2, -2.8] },
];

export function CinematicStoryCanvas({ progress, activeScene, isRTL = false }: CinematicStoryCanvasProps) {
  const mountRef = useRef<HTMLDivElement>(null);
  const progressRef = useRef(progress);
  const activeSceneRef = useRef(activeScene);
  const sceneElementsRef = useRef<{
    updateProgress: (p: number, scene: number) => void;
    dispose: () => void;
  } | null>(null);

  // Keep ref values up to date without triggering useEffect restarts
  useEffect(() => {
    progressRef.current = progress;
    activeSceneRef.current = activeScene;
    if (sceneElementsRef.current) {
      sceneElementsRef.current.updateProgress(progress, activeScene);
    }
  }, [progress, activeScene]);

  useEffect(() => {
    const container = mountRef.current;
    if (!container) return;

    let animId: number;
    let isVisible = true;
    let clock = new THREE.Clock();

    const width = container.clientWidth || window.innerWidth;
    const height = container.clientHeight || window.innerHeight;

    // 1. Scene & Fog Setup
    const scene = new THREE.Scene();
    scene.fog = new THREE.FogExp2(0x050811, 0.022);

    // 2. Camera Setup
    const camera = new THREE.PerspectiveCamera(48, width / height, 0.1, 100);
    camera.position.set(0, 3.5, 24);

    // 3. Renderer Setup
    const renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: true,
      powerPreference: "high-performance",
    });
    renderer.setSize(width, height);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    renderer.setClearColor(0x050811, 1);
    container.appendChild(renderer.domElement);

    // 4. Lighting System
    const ambientLight = new THREE.AmbientLight(0x1e293b, 1.2);
    scene.add(ambientLight);

    const dirLight = new THREE.DirectionalLight(0x38bdf8, 1.5);
    dirLight.position.set(5, 10, 7);
    scene.add(dirLight);

    // Dynamic point light at target node (storefront.production)
    const targetPointLight = new THREE.PointLight(0x06b6d4, 3.0, 20);
    targetPointLight.position.set(4.2, 0.6, -1.2);
    scene.add(targetPointLight);

    // Core point light
    const corePointLight = new THREE.PointLight(0x3b82f6, 4.0, 25);
    corePointLight.position.set(0, 0, 0);
    scene.add(corePointLight);

    // 5. Ambient Starfield Particles
    const starCount = 600;
    const starGeo = new THREE.BufferGeometry();
    const starPositions = new Float32Array(starCount * 3);
    for (let i = 0; i < starCount * 3; i += 3) {
      starPositions[i] = (Math.random() - 0.5) * 60;
      starPositions[i + 1] = (Math.random() - 0.5) * 45;
      starPositions[i + 2] = (Math.random() - 0.5) * 50;
    }
    starGeo.setAttribute("position", new THREE.BufferAttribute(starPositions, 3));
    const starMat = new THREE.PointsMaterial({
      color: 0x94a3b8,
      size: 0.08,
      transparent: true,
      opacity: 0.5,
    });
    const starField = new THREE.Points(starGeo, starMat);
    scene.add(starField);

    // 6. Central Monitoring Engine Core (at 0, 0, 0)
    const coreGroup = new THREE.Group();
    scene.add(coreGroup);

    // Outer Geodesic Icosahedron
    const coreOuterGeo = new THREE.IcosahedronGeometry(2.5, 2);
    const coreOuterMat = new THREE.MeshBasicMaterial({
      color: 0x3b82f6,
      wireframe: true,
      transparent: true,
      opacity: 0.4,
    });
    const coreOuterMesh = new THREE.Mesh(coreOuterGeo, coreOuterMat);
    coreGroup.add(coreOuterMesh);

    // Middle Crystalline Octahedron
    const coreMidGeo = new THREE.OctahedronGeometry(1.6, 1);
    const coreMidMat = new THREE.MeshBasicMaterial({
      color: 0x06b6d4,
      wireframe: true,
      transparent: true,
      opacity: 0.6,
    });
    const coreMidMesh = new THREE.Mesh(coreMidGeo, coreMidMat);
    coreGroup.add(coreMidMesh);

    // Inner Glowing Core Sphere
    const coreInnerGeo = new THREE.SphereGeometry(0.9, 24, 24);
    const coreInnerMat = new THREE.MeshBasicMaterial({
      color: 0x60a5fa,
      transparent: true,
      opacity: 0.9,
    });
    const coreInnerMesh = new THREE.Mesh(coreInnerGeo, coreInnerMat);
    coreGroup.add(coreInnerMesh);

    // Triple Gimbal Rings
    const ringMat1 = new THREE.MeshBasicMaterial({ color: 0x818cf8, transparent: true, opacity: 0.45 });
    const ringMat2 = new THREE.MeshBasicMaterial({ color: 0x38bdf8, transparent: true, opacity: 0.4 });
    const ringMat3 = new THREE.MeshBasicMaterial({ color: 0x34d399, transparent: true, opacity: 0.35 });

    const ring1 = new THREE.Mesh(new THREE.TorusGeometry(3.6, 0.03, 16, 90), ringMat1);
    ring1.rotation.x = Math.PI / 3;
    ring1.rotation.y = Math.PI / 7;
    coreGroup.add(ring1);

    const ring2 = new THREE.Mesh(new THREE.TorusGeometry(4.3, 0.03, 16, 90), ringMat2);
    ring2.rotation.x = -Math.PI / 4;
    ring2.rotation.z = Math.PI / 6;
    coreGroup.add(ring2);

    const ring3 = new THREE.Mesh(new THREE.TorusGeometry(5.0, 0.025, 16, 90), ringMat3);
    ring3.rotation.y = Math.PI / 3;
    coreGroup.add(ring3);

    // 7. Network Nodes Assembly
    const nodesGroup = new THREE.Group();
    scene.add(nodesGroup);

    interface BuiltNode {
      def: NodeDef;
      mesh: THREE.Mesh;
      halo: THREE.Mesh;
      meshMat: THREE.MeshBasicMaterial;
      haloMat: THREE.MeshBasicMaterial;
      posVec: THREE.Vector3;
    }

    const builtNodes: BuiltNode[] = [];
    let targetNodeObj: BuiltNode | null = null;

    NODES_DATA.forEach((def) => {
      const posVec = new THREE.Vector3(...def.pos);

      let baseColor = 0x10b981; // emerald default
      if (def.type === "api") baseColor = 0x38bdf8; // sky
      if (def.type === "worker") baseColor = 0xa855f7; // purple
      if (def.type === "edge") baseColor = 0x06b6d4; // cyan
      if (def.isTarget) baseColor = 0x10b981; // starts healthy

      const nodeRadius = def.isTarget ? 0.52 : 0.35;
      const nodeGeo = new THREE.SphereGeometry(nodeRadius, 20, 20);
      const meshMat = new THREE.MeshBasicMaterial({
        color: baseColor,
        transparent: true,
        opacity: 0.95,
      });
      const nodeMesh = new THREE.Mesh(nodeGeo, meshMat);
      nodeMesh.position.copy(posVec);
      nodesGroup.add(nodeMesh);

      const haloGeo = new THREE.RingGeometry(nodeRadius + 0.12, nodeRadius + 0.22, 28);
      const haloMat = new THREE.MeshBasicMaterial({
        color: baseColor,
        transparent: true,
        opacity: 0.5,
        side: THREE.DoubleSide,
      });
      const haloMesh = new THREE.Mesh(haloGeo, haloMat);
      haloMesh.position.copy(posVec);
      nodesGroup.add(haloMesh);

      const builtNode = { def, mesh: nodeMesh, halo: haloMesh, meshMat, haloMat, posVec };
      builtNodes.push(builtNode);

      if (def.isTarget) {
        targetNodeObj = builtNode;
      }
    });

    // 8. Connection Splines & Travelling Light Packets
    const splinesGroup = new THREE.Group();
    scene.add(splinesGroup);

    interface SplineConnection {
      node: BuiltNode;
      curve: THREE.CatmullRomCurve3;
      lineMesh: THREE.Line;
      lineMat: THREE.LineBasicMaterial;
      pointsCount: number;
      defaultPoints: THREE.Vector3[];
      isTargetConnection: boolean;
      packetMesh: THREE.Mesh;
      packetOffset: number;
      packetSpeed: number;
    }

    const splineConnections: SplineConnection[] = [];

    // The primary target spline connecting Core to Target Node has a special severed geometry
    let targetSplinePartA: THREE.Line | null = null;
    let targetSplinePartB: THREE.Line | null = null;
    let targetSplineMat: THREE.LineBasicMaterial | null = null;

    builtNodes.forEach((node, idx) => {
      const isTarget = !!node.def.isTarget;
      const midPoint = new THREE.Vector3(
        node.posVec.x * 0.5 + (idx % 2 === 0 ? 0.7 : -0.7),
        node.posVec.y * 0.5 + 0.5,
        node.posVec.z * 0.5 + (idx % 3 === 0 ? 0.4 : -0.4)
      );

      const curve = new THREE.CatmullRomCurve3([
        new THREE.Vector3(0, 0, 0),
        midPoint,
        node.posVec,
      ]);

      const points = curve.getPoints(40);
      const lineGeo = new THREE.BufferGeometry().setFromPoints(points);
      const lineMat = new THREE.LineBasicMaterial({
        color: isTarget ? 0x10b981 : 0x06b6d4,
        transparent: true,
        opacity: isTarget ? 0.85 : 0.35,
      });
      const lineMesh = new THREE.Line(lineGeo, lineMat);
      splinesGroup.add(lineMesh);

      // Light packet traveling along spline
      const packetGeo = new THREE.SphereGeometry(isTarget ? 0.16 : 0.1, 10, 10);
      const packetMat = new THREE.MeshBasicMaterial({
        color: 0xffffff,
        transparent: true,
        opacity: 0.95,
      });
      const packetMesh = new THREE.Mesh(packetGeo, packetMat);
      splinesGroup.add(packetMesh);

      splineConnections.push({
        node,
        curve,
        lineMesh,
        lineMat,
        pointsCount: 40,
        defaultPoints: points,
        isTargetConnection: isTarget,
        packetMesh,
        packetOffset: (idx / builtNodes.length),
        packetSpeed: 0.0035 + (idx % 3) * 0.0015,
      });

      if (isTarget) {
        targetSplineMat = lineMat;
      }
    });

    // 9. Severed Spline Spark Fragments (Scene 04 Outage Visuals)
    const sparksCount = 60;
    const sparksGeo = new THREE.BufferGeometry();
    const sparksPositions = new Float32Array(sparksCount * 3);
    const sparksVelocities = new Float32Array(sparksCount * 3);
    const sparksBasePos = new THREE.Vector3(2.1, 0.55, -0.6); // midpoint of target spline

    for (let i = 0; i < sparksCount; i++) {
      sparksPositions[i * 3] = sparksBasePos.x;
      sparksPositions[i * 3 + 1] = sparksBasePos.y;
      sparksPositions[i * 3 + 2] = sparksBasePos.z;

      // Random burst velocity
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.random() * Math.PI;
      const speed = 0.8 + Math.random() * 2.2;
      sparksVelocities[i * 3] = Math.sin(phi) * Math.cos(theta) * speed;
      sparksVelocities[i * 3 + 1] = Math.cos(phi) * speed;
      sparksVelocities[i * 3 + 2] = Math.sin(phi) * Math.sin(theta) * speed;
    }
    sparksGeo.setAttribute("position", new THREE.BufferAttribute(sparksPositions, 3));
    const sparksMat = new THREE.PointsMaterial({
      color: 0xff3b30,
      size: 0.14,
      transparent: true,
      opacity: 0,
    });
    const sparksField = new THREE.Points(sparksGeo, sparksMat);
    scene.add(sparksField);

    // 10. Outage Shockwave System (Scene 04 Visual Centerpiece)
    const shockwaveGroup = new THREE.Group();
    shockwaveGroup.position.set(4.2, 0.6, -1.2);
    scene.add(shockwaveGroup);

    // Primary Shockwave Ring
    const shockwaveGeo1 = new THREE.RingGeometry(0.1, 0.28, 64);
    const shockwaveMat1 = new THREE.MeshBasicMaterial({
      color: 0xef4444,
      transparent: true,
      opacity: 0,
      side: THREE.DoubleSide,
    });
    const shockwaveMesh1 = new THREE.Mesh(shockwaveGeo1, shockwaveMat1);
    shockwaveGroup.add(shockwaveMesh1);

    // Secondary Shockwave Halo Ring
    const shockwaveGeo2 = new THREE.RingGeometry(0.32, 0.42, 64);
    const shockwaveMat2 = new THREE.MeshBasicMaterial({
      color: 0xf87171,
      transparent: true,
      opacity: 0,
      side: THREE.DoubleSide,
    });
    const shockwaveMesh2 = new THREE.Mesh(shockwaveGeo2, shockwaveMat2);
    shockwaveGroup.add(shockwaveMesh2);

    // 11. Restorative Wave (Scene 06 Visual Conclusion)
    const recoveryWaveGeo = new THREE.RingGeometry(0.1, 0.4, 64);
    const recoveryWaveMat = new THREE.MeshBasicMaterial({
      color: 0x10b981,
      transparent: true,
      opacity: 0,
      side: THREE.DoubleSide,
    });
    const recoveryWaveMesh = new THREE.Mesh(recoveryWaveGeo, recoveryWaveMat);
    recoveryWaveMesh.position.set(4.2, 0.6, -1.2);
    scene.add(recoveryWaveMesh);

    // 12. Camera Path Spline: 6 Choreographed Narrative Waypoints
    // Waypoint indices:
    // 0: Scene 01 Connected (wide establishing view)
    // 1: Scene 02 Active Probing (dolly forward tracking data streams)
    // 2: Scene 03 Degradation (approaching failing region, amber tilt)
    // 3: Scene 04 Outage (close dramatic orbit around failed node)
    // 4: Scene 05 Incident Awareness (tactical pull-back isometric angle)
    // 5: Scene 06 Recovery (soaring through restored network into next section)
    const cameraWaypoints = [
      { pos: new THREE.Vector3(0, 2.5, 22), lookAt: new THREE.Vector3(0, 0, 0) },
      { pos: new THREE.Vector3(2.0, 1.4, 15), lookAt: new THREE.Vector3(2.5, 0.4, 0) },
      { pos: new THREE.Vector3(3.6, 1.8, 8.5), lookAt: new THREE.Vector3(4.2, 0.6, -1.2) },
      { pos: new THREE.Vector3(4.8, 0.4, 4.2), lookAt: new THREE.Vector3(4.2, 0.6, -1.2) },
      { pos: new THREE.Vector3(1.2, 5.8, 14.5), lookAt: new THREE.Vector3(3.0, 0.2, -0.8) },
      { pos: new THREE.Vector3(0, 1.2, 16), lookAt: new THREE.Vector3(0, 0.2, 0) },
    ];

    // Helper: interpolate between two Vector3s
    const currentCamPos = new THREE.Vector3();
    const currentLookAt = new THREE.Vector3();

    function updateCameraFromProgress(p: number) {
      // p is [0, 1]
      // Map to 5 segments across the 6 waypoints
      const segments = cameraWaypoints.length - 1;
      const scaledP = p * segments;
      const segIndex = Math.min(Math.floor(scaledP), segments - 1);
      const segFraction = scaledP - segIndex;

      // Smooth step easing
      const t = segFraction * segFraction * (3 - 2 * segFraction);

      const wpStart = cameraWaypoints[segIndex];
      const wpEnd = cameraWaypoints[segIndex + 1];

      currentCamPos.lerpVectors(wpStart.pos, wpEnd.pos, t);
      currentLookAt.lerpVectors(wpStart.lookAt, wpEnd.lookAt, t);

      camera.position.copy(currentCamPos);
      camera.lookAt(currentLookAt);
    }

    // Function to update Three.js 3D elements based on scrubbed scroll progress
    function updateProgress(p: number, sceneIdx: number) {
      updateCameraFromProgress(p);

      // =========================================================================
      // SCENE 01: CONNECTED (p ~ 0.00 to 0.16)
      // Pristine emerald & cyan harmony, core activates
      // =========================================================================
      if (p < 0.16) {
        const subP = p / 0.16;
        ambientLight.color.setHex(0x1e293b);
        ambientLight.intensity = 1.0 + subP * 0.4;
        corePointLight.color.setHex(0x3b82f6);
        targetPointLight.color.setHex(0x10b981);
        targetPointLight.intensity = 2.0;

        if (targetNodeObj) {
          targetNodeObj.meshMat.color.setHex(0x10b981);
          targetNodeObj.haloMat.color.setHex(0x10b981);
          targetNodeObj.haloMat.opacity = 0.5;
        }

        if (targetSplineMat) {
          targetSplineMat.color.setHex(0x10b981);
          targetSplineMat.opacity = 0.85;
        }

        shockwaveMat1.opacity = 0;
        shockwaveMat2.opacity = 0;
        sparksMat.opacity = 0;
        recoveryWaveMat.opacity = 0;
      }

      // =========================================================================
      // SCENE 02: ACTIVE TELEMETRY (p ~ 0.16 to 0.35)
      // Fast probe pulses, cyan data streams, network in high activity
      // =========================================================================
      else if (p >= 0.16 && p < 0.35) {
        const subP = (p - 0.16) / 0.19;
        ambientLight.color.setHex(0x0f172a);
        targetPointLight.color.setHex(0x06b6d4);
        targetPointLight.intensity = 2.5 + Math.sin(subP * Math.PI) * 1.5;

        if (targetNodeObj) {
          targetNodeObj.meshMat.color.setHex(0x06b6d4);
          targetNodeObj.haloMat.color.setHex(0x06b6d4);
          targetNodeObj.haloMat.opacity = 0.6 + Math.sin(subP * 8) * 0.2;
        }

        if (targetSplineMat) {
          targetSplineMat.color.setHex(0x06b6d4);
          targetSplineMat.opacity = 0.9;
        }

        shockwaveMat1.opacity = 0;
        shockwaveMat2.opacity = 0;
        sparksMat.opacity = 0;
        recoveryWaveMat.opacity = 0;
      }

      // =========================================================================
      // SCENE 03: DEGRADATION (p ~ 0.35 to 0.52)
      // Amber warning lights, jittering spline vertices, packet instability
      // =========================================================================
      else if (p >= 0.35 && p < 0.52) {
        const subP = (p - 0.35) / 0.17;
        ambientLight.color.setHex(0x291807);
        targetPointLight.color.setHex(0xf59e0b);
        targetPointLight.intensity = 3.5 + Math.sin(subP * 12) * 2.0;

        if (targetNodeObj) {
          targetNodeObj.meshMat.color.setHex(0xf59e0b);
          targetNodeObj.haloMat.color.setHex(0xf59e0b);
          targetNodeObj.haloMat.opacity = 0.7 + Math.sin(subP * 15) * 0.25;
        }

        if (targetSplineMat) {
          targetSplineMat.color.setHex(0xf59e0b);
          targetSplineMat.opacity = 0.95;
        }

        // Add dynamic high-frequency jitter to target spline points
        const targetConn = splineConnections.find((c) => c.isTargetConnection);
        if (targetConn) {
          const posAttr = targetConn.lineMesh.geometry.attributes.position;
          const posArr = posAttr.array as Float32Array;
          for (let i = 0; i < targetConn.pointsCount; i++) {
            const defPt = targetConn.defaultPoints[i];
            const jitterFactor = Math.sin(i * 0.8 + subP * 20) * subP * 0.28;
            posArr[i * 3] = defPt.x + jitterFactor;
            posArr[i * 3 + 1] = defPt.y + (Math.cos(i + subP * 15) * subP * 0.2);
            posArr[i * 3 + 2] = defPt.z + jitterFactor * 0.5;
          }
          posAttr.needsUpdate = true;
        }

        shockwaveMat1.opacity = 0;
        shockwaveMat2.opacity = 0;
        sparksMat.opacity = 0;
        recoveryWaveMat.opacity = 0;
      }

      // =========================================================================
      // SCENE 04: TOTAL OUTAGE (p ~ 0.52 to 0.68)
      // Severed connection! Visible expanding shockwave! Red warning beacon!
      // =========================================================================
      else if (p >= 0.52 && p < 0.68) {
        const subP = (p - 0.52) / 0.16;
        ambientLight.color.setHex(0x380b0b);
        ambientLight.intensity = 1.4;
        targetPointLight.color.setHex(0xef4444);
        targetPointLight.intensity = 7.0 + Math.sin(subP * 16) * 3.0; // pulsing strobe

        if (targetNodeObj) {
          targetNodeObj.meshMat.color.setHex(0xef4444);
          targetNodeObj.haloMat.color.setHex(0xff3b30);
          targetNodeObj.haloMat.opacity = 0.9;
        }

        // Physically sever the target spline: hide line or collapse its midpoint
        const targetConn = splineConnections.find((c) => c.isTargetConnection);
        if (targetConn) {
          const posAttr = targetConn.lineMesh.geometry.attributes.position;
          const posArr = posAttr.array as Float32Array;
          const midIdx = Math.floor(targetConn.pointsCount / 2);
          for (let i = 0; i < targetConn.pointsCount; i++) {
            const defPt = targetConn.defaultPoints[i];
            // Split line into two disjoint stubs with a wide empty gulf in the middle
            if (i >= midIdx - 8 && i <= midIdx + 8) {
              posArr[i * 3] = NaN; // creates a visible gap in WebGL line rendering
              posArr[i * 3 + 1] = NaN;
              posArr[i * 3 + 2] = NaN;
            } else {
              posArr[i * 3] = defPt.x;
              posArr[i * 3 + 1] = defPt.y;
              posArr[i * 3 + 2] = defPt.z;
            }
          }
          posAttr.needsUpdate = true;
          targetConn.lineMat.color.setHex(0xef4444);
          targetConn.lineMat.opacity = 0.7;
        }

        // Propagate expanding shockwave ring
        const shockwaveScale = 0.5 + subP * 14.0;
        shockwaveGroup.scale.set(shockwaveScale, shockwaveScale, shockwaveScale);
        shockwaveMesh1.lookAt(camera.position);
        shockwaveMesh2.lookAt(camera.position);
        shockwaveMat1.opacity = Math.max(0, 0.9 - subP * 0.9);
        shockwaveMat2.opacity = Math.max(0, 0.7 - subP * 0.7);

        // Scatter severed sparks outwards
        sparksMat.opacity = Math.max(0, 1.0 - subP * 0.85);
        const sparksPosAttr = sparksGeo.attributes.position;
        const sparksPosArr = sparksPosAttr.array as Float32Array;
        for (let i = 0; i < sparksCount; i++) {
          sparksPosArr[i * 3] = sparksBasePos.x + sparksVelocities[i * 3] * subP * 3.5;
          sparksPosArr[i * 3 + 1] = sparksBasePos.y + sparksVelocities[i * 3 + 1] * subP * 3.5;
          sparksPosArr[i * 3 + 2] = sparksBasePos.z + sparksVelocities[i * 3 + 2] * subP * 3.5;
        }
        sparksPosAttr.needsUpdate = true;

        recoveryWaveMat.opacity = 0;
      }

      // =========================================================================
      // SCENE 05: INCIDENT AWARENESS (p ~ 0.68 to 0.84)
      // Tactical isometric view, violet tactical grid, stable incident state
      // =========================================================================
      else if (p >= 0.68 && p < 0.84) {
        const subP = (p - 0.68) / 0.16;
        ambientLight.color.setHex(0x1e1536);
        targetPointLight.color.setHex(0x8b5cf6);
        targetPointLight.intensity = 4.0;

        if (targetNodeObj) {
          targetNodeObj.meshMat.color.setHex(0x8b5cf6);
          targetNodeObj.haloMat.color.setHex(0x8b5cf6);
          targetNodeObj.haloMat.opacity = 0.75;
        }

        // Spline is ready to recover
        const targetConn = splineConnections.find((c) => c.isTargetConnection);
        if (targetConn) {
          targetConn.lineMat.color.setHex(0x8b5cf6);
          targetConn.lineMat.opacity = 0.5;
        }

        shockwaveMat1.opacity = 0;
        shockwaveMat2.opacity = 0;
        sparksMat.opacity = 0;
        recoveryWaveMat.opacity = 0;
      }

      // =========================================================================
      // SCENE 06: RECOVERY & RESOLUTION (p ~ 0.84 to 1.00)
      // Restorative wave sweeps, severed connection re-knits into emerald
      // =========================================================================
      else {
        const subP = (p - 0.84) / 0.16;
        ambientLight.color.setHex(0x06281e);
        ambientLight.intensity = 1.3;
        targetPointLight.color.setHex(0x10b981);
        targetPointLight.intensity = 3.5 + (1 - subP) * 3.0;

        if (targetNodeObj) {
          targetNodeObj.meshMat.color.setHex(0x10b981);
          targetNodeObj.haloMat.color.setHex(0x34d399);
          targetNodeObj.haloMat.opacity = 0.8;
        }

        // Reconnect and restore target spline fully
        const targetConn = splineConnections.find((c) => c.isTargetConnection);
        if (targetConn) {
          const posAttr = targetConn.lineMesh.geometry.attributes.position;
          const posArr = posAttr.array as Float32Array;
          for (let i = 0; i < targetConn.pointsCount; i++) {
            const defPt = targetConn.defaultPoints[i];
            posArr[i * 3] = defPt.x;
            posArr[i * 3 + 1] = defPt.y;
            posArr[i * 3 + 2] = defPt.z;
          }
          posAttr.needsUpdate = true;
          targetConn.lineMat.color.setHex(0x10b981);
          targetConn.lineMat.opacity = 0.9;
        }

        // Restorative Wave Expansion
        const recScale = 0.5 + subP * 12.0;
        recoveryWaveMesh.scale.set(recScale, recScale, recScale);
        recoveryWaveMesh.lookAt(camera.position);
        recoveryWaveMat.opacity = Math.max(0, 0.85 - subP * 0.85);

        shockwaveMat1.opacity = 0;
        shockwaveMat2.opacity = 0;
        sparksMat.opacity = 0;
      }
    }

    // 13. High-Performance Render Loop
    function animate() {
      animId = requestAnimationFrame(animate);
      if (!isVisible) return;

      const delta = clock.getDelta();
      const time = clock.getElapsedTime();

      // Slow idle rotation of core gimbal rings
      coreOuterMesh.rotation.y += 0.003;
      coreOuterMesh.rotation.x += 0.002;
      coreMidMesh.rotation.y -= 0.004;
      coreMidMesh.rotation.z += 0.003;

      ring1.rotation.z += 0.006;
      ring2.rotation.y -= 0.005;
      ring3.rotation.x += 0.004;

      // Node subtle breathing and halo camera alignment
      builtNodes.forEach((node) => {
        node.halo.lookAt(camera.position);
      });

      // Animate packet movements along curves
      splineConnections.forEach((conn) => {
        // If outage active, stop packets on target connection
        if (conn.isTargetConnection && progressRef.current >= 0.52 && progressRef.current < 0.84) {
          conn.packetMesh.visible = false;
        } else {
          conn.packetMesh.visible = true;
          conn.packetOffset = (conn.packetOffset + conn.packetSpeed) % 1.0;
          const point = conn.curve.getPointAt(conn.packetOffset);
          conn.packetMesh.position.copy(point);
        }
      });

      renderer.render(scene, camera);
    }

    // Initial position
    updateProgress(progressRef.current, activeSceneRef.current);
    animate();

    // 14. Responsive Resize Observer
    const handleResize = () => {
      if (!container) return;
      const w = container.clientWidth || window.innerWidth;
      const h = container.clientHeight || window.innerHeight;
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
      renderer.setSize(w, h);
    };

    window.addEventListener("resize", handleResize);

    // 15. Intersection Observer to Pause Render Loop when Offscreen
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          isVisible = entry.isIntersecting;
        });
      },
      { threshold: 0.05 }
    );
    observer.observe(container);

    sceneElementsRef.current = {
      updateProgress,
      dispose: () => {
        cancelAnimationFrame(animId);
        window.removeEventListener("resize", handleResize);
        observer.disconnect();

        // Strict GPU disposal
        renderer.dispose();
        if (renderer.domElement.parentElement) {
          renderer.domElement.parentElement.removeChild(renderer.domElement);
        }

        coreOuterGeo.dispose();
        coreOuterMat.dispose();
        coreMidGeo.dispose();
        coreMidMat.dispose();
        coreInnerGeo.dispose();
        coreInnerMat.dispose();
        ringMat1.dispose();
        ringMat2.dispose();
        ringMat3.dispose();
        starGeo.dispose();
        starMat.dispose();
        shockwaveGeo1.dispose();
        shockwaveMat1.dispose();
        shockwaveGeo2.dispose();
        shockwaveMat2.dispose();
        recoveryWaveGeo.dispose();
        recoveryWaveMat.dispose();
        sparksGeo.dispose();
        sparksMat.dispose();

        builtNodes.forEach((n) => {
          n.mesh.geometry.dispose();
          n.meshMat.dispose();
          n.halo.geometry.dispose();
          n.haloMat.dispose();
        });

        splineConnections.forEach((s) => {
          s.lineMesh.geometry.dispose();
          s.lineMat.dispose();
          s.packetMesh.geometry.dispose();
          (s.packetMesh.material as THREE.Material).dispose();
        });
      },
    };

    return () => {
      if (sceneElementsRef.current) {
        sceneElementsRef.current.dispose();
      }
    };
  }, []);

  return (
    <div
      ref={mountRef}
      className="absolute inset-0 w-full h-full pointer-events-none overflow-hidden"
      style={{ zIndex: 1 }}
    />
  );
}

