"use client";

import React, { useEffect, useRef } from "react";
import * as THREE from "three";

interface CinematicStoryCanvasProps {
  progress: number; // 0.0 to 1.0 (scrubbed by GSAP ScrollTrigger)
  activeScene: number; // 0 to 5
  isRTL?: boolean;
}

interface RegionalSpire {
  id: string;
  name: string;
  region: string;
  pos: [number, number, number];
  isTarget?: boolean;
  color: number;
}

const REGIONAL_SPIRES: RegionalSpire[] = [
  { id: "target-store", name: "storefront.production", region: "EU-Central (Primary)", pos: [4.2, 0.6, -1.2], isTarget: true, color: 0x10b981 },
  { id: "spire-us", name: "api.gateway.us-east", region: "US-East (Virginia)", pos: [-3.8, 1.8, -0.5], color: 0x38bdf8 },
  { id: "spire-ap", name: "probe.singapore.edge", region: "AP-East (Singapore)", pos: [-2.4, -1.5, 2.2], color: 0x06b6d4 },
  { id: "spire-me", name: "edge.tehran.core", region: "ME-Central (Tehran)", pos: [5.2, -1.4, 2.0], color: 0x34d399 },
  { id: "spire-fra", name: "cdn.edge.frankfurt", region: "EU-West (Frankfurt)", pos: [2.5, 2.8, 1.8], color: 0x38bdf8 },
  { id: "spire-tok", name: "probe.tokyo.edge", region: "AP-Northeast (Tokyo)", pos: [-4.6, 0.4, -2.8], color: 0x818cf8 },
];

export function CinematicStoryCanvas({ progress, activeScene, isRTL = false }: CinematicStoryCanvasProps) {
  const mountRef = useRef<HTMLDivElement>(null);
  const progressRef = useRef(progress);
  const activeSceneRef = useRef(activeScene);
  const sceneElementsRef = useRef<{
    updateProgress: (p: number, scene: number) => void;
    dispose: () => void;
  } | null>(null);

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
    const clock = new THREE.Clock();

    const width = container.clientWidth || window.innerWidth;
    const height = container.clientHeight || window.innerHeight;

    // 1. SCENE GRAPH & ATMOSPHERIC FOG
    const scene = new THREE.Scene();
    scene.fog = new THREE.FogExp2(0x03050c, 0.022);

    // 2. CAMERA SETUP
    const camera = new THREE.PerspectiveCamera(48, width / height, 0.1, 120);
    camera.position.set(0, 3.5, 26);

    // 3. WEBGL RENDERER
    const renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: true,
      powerPreference: "high-performance",
    });
    renderer.setSize(width, height);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    renderer.setClearColor(0x03050c, 1);
    renderer.toneMapping = THREE.ACESFilmicToneMapping;
    renderer.toneMappingExposure = 1.15;
    container.appendChild(renderer.domElement);

    // 4. LIGHTING ECOSYSTEM
    const ambientLight = new THREE.AmbientLight(0x0f172a, 1.2);
    scene.add(ambientLight);

    const keyDirectional = new THREE.DirectionalLight(0x38bdf8, 1.8);
    keyDirectional.position.set(8, 14, 10);
    scene.add(keyDirectional);

    // Dynamic Central Reactor Core Light
    const reactorLight = new THREE.PointLight(0x10b981, 4.5, 30);
    reactorLight.position.set(0, 0.5, 0);
    scene.add(reactorLight);

    // Target Node Spotlight / PointLight
    const targetLight = new THREE.PointLight(0x10b981, 3.2, 18);
    targetLight.position.set(4.2, 0.6, -1.2);
    scene.add(targetLight);

    // Emergency Outage Strobe Light (disabled until Scene 04)
    const emergencyStrobeLight = new THREE.PointLight(0xef4444, 0, 40);
    emergencyStrobeLight.position.set(3.5, 2.5, 0);
    scene.add(emergencyStrobeLight);

    // 5. CYBERNETIC PERSPECTIVE GRID FLOOR
    const gridHelper = new THREE.GridHelper(70, 70, 0x0ea5e9, 0x1e293b);
    gridHelper.position.y = -3.2;
    (gridHelper.material as THREE.Material).transparent = true;
    (gridHelper.material as THREE.Material).opacity = 0.35;
    scene.add(gridHelper);

    // Additional glowing runway lines along Z-axis
    const runwayGeo = new THREE.BufferGeometry();
    const runwayPositions = new Float32Array([
      -3, -3.18, -35,   -3, -3.18, 35,
       3, -3.18, -35,    3, -3.18, 35,
      -8, -3.18, -35,   -8, -3.18, 35,
       8, -3.18, -35,    8, -3.18, 35,
    ]);
    runwayGeo.setAttribute("position", new THREE.BufferAttribute(runwayPositions, 3));
    const runwayMat = new THREE.LineBasicMaterial({ color: 0x38bdf8, transparent: true, opacity: 0.5 });
    const runwayLines = new THREE.LineSegments(runwayGeo, runwayMat);
    scene.add(runwayLines);

    // 6. PERIMETER STRUCTURAL MONOLITH COLUMNS
    const columnsGroup = new THREE.Group();
    scene.add(columnsGroup);
    const colGeo = new THREE.BoxGeometry(1.2, 16, 1.2);
    const colMat = new THREE.MeshStandardMaterial({
      color: 0x090d16,
      roughness: 0.25,
      metalness: 0.85,
    });
    const colGlowMat = new THREE.MeshBasicMaterial({ color: 0x0284c7, transparent: true, opacity: 0.6 });

    const columnCoords: [number, number][] = [
      [-12, -20], [-12, -8], [-12, 6], [-12, 20],
      [ 12, -20], [ 12, -8], [ 12, 6], [ 12, 20],
    ];

    columnCoords.forEach(([x, z]) => {
      const colMesh = new THREE.Mesh(colGeo, colMat);
      colMesh.position.set(x, 4.0, z);
      columnsGroup.add(colMesh);

      // Vertical neon indicator strip
      const stripGeo = new THREE.BoxGeometry(0.08, 15.5, 0.08);
      const stripMesh = new THREE.Mesh(stripGeo, colGlowMat);
      stripMesh.position.set(x > 0 ? x - 0.62 : x + 0.62, 4.0, z);
      columnsGroup.add(stripMesh);
    });

    // 7. KINETIC MONOLITH ARMOR GATES (SCENE 00 / 01 HULL BREACH)
    const gatesGroup = new THREE.Group();
    gatesGroup.position.set(0, 1.5, 18);
    scene.add(gatesGroup);

    const gatePlateGeo = new THREE.BoxGeometry(8.5, 14, 0.6);
    const gatePlateMat = new THREE.MeshStandardMaterial({
      color: 0x070a12,
      roughness: 0.2,
      metalness: 0.9,
    });
    const gateBevelMat = new THREE.MeshBasicMaterial({ color: 0x38bdf8, transparent: true, opacity: 0.8 });

    // Left Armor Plate
    const leftPlate = new THREE.Mesh(gatePlateGeo, gatePlateMat);
    leftPlate.position.set(-4.3, 0, 0);
    gatesGroup.add(leftPlate);
    const leftBevel = new THREE.Mesh(new THREE.BoxGeometry(0.12, 13.8, 0.65), gateBevelMat);
    leftBevel.position.set(-0.05, 0, 0);
    leftPlate.add(leftBevel);

    // Right Armor Plate
    const rightPlate = new THREE.Mesh(gatePlateGeo, gatePlateMat);
    rightPlate.position.set(4.3, 0, 0);
    gatesGroup.add(rightPlate);
    const rightBevel = new THREE.Mesh(new THREE.BoxGeometry(0.12, 13.8, 0.65), gateBevelMat);
    rightBevel.position.set(0.05, 0, 0);
    rightPlate.add(rightBevel);

    // 8. CENTRAL RELIABILITY REACTOR CORE (at 0, 0.5, 0)
    const reactorGroup = new THREE.Group();
    reactorGroup.position.set(0, 0.5, 0);
    scene.add(reactorGroup);

    // Cylindrical Magnetic Stator Cage
    const cageGeo = new THREE.CylinderGeometry(3.0, 3.0, 2.4, 16, 1, true);
    const cageMat = new THREE.MeshBasicMaterial({
      color: 0x1e293b,
      wireframe: true,
      transparent: true,
      opacity: 0.45,
    });
    const cageMesh = new THREE.Mesh(cageGeo, cageMat);
    reactorGroup.add(cageMesh);

    // Rotating Gimbal Rings
    const ring1 = new THREE.Mesh(
      new THREE.TorusGeometry(3.8, 0.04, 16, 80),
      new THREE.MeshBasicMaterial({ color: 0x38bdf8, transparent: true, opacity: 0.6 })
    );
    ring1.rotation.x = Math.PI / 3;
    reactorGroup.add(ring1);

    const ring2 = new THREE.Mesh(
      new THREE.TorusGeometry(4.6, 0.035, 16, 80),
      new THREE.MeshBasicMaterial({ color: 0x10b981, transparent: true, opacity: 0.5 })
    );
    ring2.rotation.y = Math.PI / 4;
    ring2.rotation.z = Math.PI / 6;
    reactorGroup.add(ring2);

    // Glowing Inner Plasma Core
    const plasmaCoreGeo = new THREE.IcosahedronGeometry(1.2, 2);
    const plasmaCoreMat = new THREE.MeshBasicMaterial({
      color: 0x34d399,
      wireframe: true,
      transparent: true,
      opacity: 0.85,
    });
    const plasmaCoreMesh = new THREE.Mesh(plasmaCoreGeo, plasmaCoreMat);
    reactorGroup.add(plasmaCoreMesh);

    // 9. REGIONAL PROBE SPIRES & BEACONS
    const spiresGroup = new THREE.Group();
    scene.add(spiresGroup);

    interface BuiltSpire {
      def: RegionalSpire;
      group: THREE.Group;
      beaconMesh: THREE.Mesh;
      haloMesh: THREE.Mesh;
      pedestalMesh: THREE.Mesh;
      beaconMat: THREE.MeshBasicMaterial;
      haloMat: THREE.MeshBasicMaterial;
    }

    const builtSpires: BuiltSpire[] = [];
    let targetSpireObj: BuiltSpire | null = null;

    REGIONAL_SPIRES.forEach((def) => {
      const spireGrp = new THREE.Group();
      spireGrp.position.set(...def.pos);
      spiresGroup.add(spireGrp);

      // Pedestal base
      const pedGeo = new THREE.CylinderGeometry(0.35, 0.55, 1.2, 12);
      const pedMat = new THREE.MeshStandardMaterial({
        color: 0x0f172a,
        roughness: 0.3,
        metalness: 0.8,
      });
      const pedestalMesh = new THREE.Mesh(pedGeo, pedMat);
      pedestalMesh.position.y = -0.6;
      spireGrp.add(pedestalMesh);

      // Glowing Beacon Top
      const beaconRadius = def.isTarget ? 0.55 : 0.38;
      const beaconGeo = new THREE.OctahedronGeometry(beaconRadius, 0);
      const beaconMat = new THREE.MeshBasicMaterial({
        color: def.color,
        transparent: true,
        opacity: 0.95,
      });
      const beaconMesh = new THREE.Mesh(beaconGeo, beaconMat);
      spireGrp.add(beaconMesh);

      // Pulsing Halo Ring
      const haloGeo = new THREE.RingGeometry(beaconRadius + 0.15, beaconRadius + 0.28, 32);
      const haloMat = new THREE.MeshBasicMaterial({
        color: def.color,
        transparent: true,
        opacity: 0.5,
        side: THREE.DoubleSide,
      });
      const haloMesh = new THREE.Mesh(haloGeo, haloMat);
      spireGrp.add(haloMesh);

      const built = { def, group: spireGrp, beaconMesh, haloMesh, pedestalMesh, beaconMat, haloMat };
      builtSpires.push(built);

      if (def.isTarget) targetSpireObj = built;
    });

    // 10. LASER-OPTIC WAVEGUIDES & PACKET STREAMS
    const waveguidesGroup = new THREE.Group();
    scene.add(waveguidesGroup);

    interface WaveguideConn {
      spire: BuiltSpire;
      curve: THREE.CatmullRomCurve3;
      lineMesh: THREE.Line;
      lineMat: THREE.LineBasicMaterial;
      defaultPoints: THREE.Vector3[];
      packetMesh: THREE.Mesh;
      packetOffset: number;
      packetSpeed: number;
    }

    const waveguides: WaveguideConn[] = [];
    let targetWaveguide: WaveguideConn | null = null;

    builtSpires.forEach((spire, idx) => {
      const isTarget = !!spire.def.isTarget;
      const midX = spire.def.pos[0] * 0.5 + (idx % 2 === 0 ? 0.6 : -0.6);
      const midY = spire.def.pos[1] * 0.5 + 0.4;
      const midZ = spire.def.pos[2] * 0.5 + (idx % 3 === 0 ? 0.5 : -0.5);

      const curve = new THREE.CatmullRomCurve3([
        new THREE.Vector3(0, 0.5, 0),
        new THREE.Vector3(midX, midY, midZ),
        new THREE.Vector3(...spire.def.pos),
      ]);

      const points = curve.getPoints(50);
      const lineGeo = new THREE.BufferGeometry().setFromPoints(points);
      const lineMat = new THREE.LineBasicMaterial({
        color: isTarget ? 0x10b981 : 0x0ea5e9,
        transparent: true,
        opacity: isTarget ? 0.9 : 0.4,
      });
      const lineMesh = new THREE.Line(lineGeo, lineMat);
      waveguidesGroup.add(lineMesh);

      // Light packet
      const packetGeo = new THREE.SphereGeometry(isTarget ? 0.16 : 0.1, 12, 12);
      const packetMat = new THREE.MeshBasicMaterial({ color: 0xffffff, transparent: true, opacity: 0.95 });
      const packetMesh = new THREE.Mesh(packetGeo, packetMat);
      waveguidesGroup.add(packetMesh);

      const wg = {
        spire,
        curve,
        lineMesh,
        lineMat,
        defaultPoints: points,
        packetMesh,
        packetOffset: idx / builtSpires.length,
        packetSpeed: 0.004 + (idx % 3) * 0.002,
      };

      waveguides.push(wg);
      if (isTarget) targetWaveguide = wg;
    });

    // 11. OUTAGE CATASTROPHE & SEVERED ARC SPARK BURST
    const sparksCount = 80;
    const sparksGeo = new THREE.BufferGeometry();
    const sparksPositions = new Float32Array(sparksCount * 3);
    const sparksVelocities = new Float32Array(sparksCount * 3);
    const sparksOrigin = new THREE.Vector3(2.1, 0.55, -0.6); // midpoint of target waveguide

    for (let i = 0; i < sparksCount; i++) {
      sparksPositions[i * 3] = sparksOrigin.x;
      sparksPositions[i * 3 + 1] = sparksOrigin.y;
      sparksPositions[i * 3 + 2] = sparksOrigin.z;

      const theta = Math.random() * Math.PI * 2;
      const phi = Math.random() * Math.PI;
      const speed = 1.0 + Math.random() * 3.0;
      sparksVelocities[i * 3] = Math.sin(phi) * Math.cos(theta) * speed;
      sparksVelocities[i * 3 + 1] = Math.cos(phi) * speed;
      sparksVelocities[i * 3 + 2] = Math.sin(phi) * Math.sin(theta) * speed;
    }
    sparksGeo.setAttribute("position", new THREE.BufferAttribute(sparksPositions, 3));
    const sparksMat = new THREE.PointsMaterial({
      color: 0xff2222,
      size: 0.18,
      transparent: true,
      opacity: 0,
    });
    const sparksMesh = new THREE.Points(sparksGeo, sparksMat);
    scene.add(sparksMesh);

    // 12. DUAL-RING OUTAGE SHOCKWAVE SYSTEM
    const shockwaveGroup = new THREE.Group();
    shockwaveGroup.position.set(4.2, 0.6, -1.2);
    scene.add(shockwaveGroup);

    const shockwaveMat1 = new THREE.MeshBasicMaterial({ color: 0xef4444, transparent: true, opacity: 0, side: THREE.DoubleSide });
    const shockwaveMesh1 = new THREE.Mesh(new THREE.RingGeometry(0.2, 0.45, 64), shockwaveMat1);
    shockwaveGroup.add(shockwaveMesh1);

    const shockwaveMat2 = new THREE.MeshBasicMaterial({ color: 0xf87171, transparent: true, opacity: 0, side: THREE.DoubleSide });
    const shockwaveMesh2 = new THREE.Mesh(new THREE.RingGeometry(0.5, 0.65, 64), shockwaveMat2);
    shockwaveGroup.add(shockwaveMesh2);

    // 13. RESTORATIVE RECOVERY HARMONIC WAVE
    const recoveryWaveMat = new THREE.MeshBasicMaterial({ color: 0x10b981, transparent: true, opacity: 0, side: THREE.DoubleSide });
    const recoveryWaveMesh = new THREE.Mesh(new THREE.RingGeometry(0.2, 0.5, 64), recoveryWaveMat);
    recoveryWaveMesh.position.set(4.2, 0.6, -1.2);
    scene.add(recoveryWaveMesh);

    // 14. CHOREOGRAPHED 3D CAMERA WAYPOINTS (7 DISTINCT PERSPECTIVES)
    const cameraWaypoints = [
      // 0: Monolith Gateway Exterior (Hero view)
      { pos: new THREE.Vector3(0, 3.2, 26), lookAt: new THREE.Vector3(0, 1.2, 0), fov: 48 },
      // 1: Supersonic descent through opening armor gates
      { pos: new THREE.Vector3(1.6, 2.0, 14), lookAt: new THREE.Vector3(0, 0.5, 0), fov: 44 },
      // 2: Orbiting low-angle sweep across regional probe spires
      { pos: new THREE.Vector3(3.2, 1.2, 8.5), lookAt: new THREE.Vector3(3.6, 0.5, 0), fov: 40 },
      // 3: Approaching failing target node (warning phase)
      { pos: new THREE.Vector3(4.0, 1.0, 4.8), lookAt: new THREE.Vector3(4.2, 0.6, -1.2), fov: 38 },
      // 4: Dramatic close orbit around severed node (outage rupture)
      { pos: new THREE.Vector3(4.8, 0.4, 3.2), lookAt: new THREE.Vector3(4.2, 0.6, -1.2), fov: 42 },
      // 5: Elevated tactical isometric vantage point
      { pos: new THREE.Vector3(1.5, 6.2, 12.0), lookAt: new THREE.Vector3(2.5, 0.2, -0.5), fov: 45 },
      // 6: Majestic rising pull-back into restored equilibrium
      { pos: new THREE.Vector3(0, 2.8, 17.0), lookAt: new THREE.Vector3(0, 0.5, 0), fov: 52 },
    ];

    const currentCamPos = new THREE.Vector3();
    const currentLookAt = new THREE.Vector3();
    let shakeIntensity = 0;

    function updateCameraSpline(p: number) {
      const segments = cameraWaypoints.length - 1;
      const scaledP = Math.max(0, Math.min(1, p)) * segments;
      const segIndex = Math.min(Math.floor(scaledP), segments - 1);
      const segFraction = scaledP - segIndex;
      const t = segFraction * segFraction * (3 - 2 * segFraction); // cubic ease

      const wpStart = cameraWaypoints[segIndex];
      const wpEnd = cameraWaypoints[segIndex + 1];

      currentCamPos.lerpVectors(wpStart.pos, wpEnd.pos, t);
      currentLookAt.lerpVectors(wpStart.lookAt, wpEnd.lookAt, t);

      // Apply seismic shake during outage (Scene 04)
      if (shakeIntensity > 0) {
        const time = clock.getElapsedTime();
        currentCamPos.x += Math.sin(time * 50) * shakeIntensity;
        currentCamPos.y += Math.cos(time * 42) * shakeIntensity;
      }

      camera.position.copy(currentCamPos);
      camera.lookAt(currentLookAt);

      const targetFov = wpStart.fov + (wpEnd.fov - wpStart.fov) * t;
      if (Math.abs(camera.fov - targetFov) > 0.1) {
        camera.fov = targetFov;
        camera.updateProjectionMatrix();
      }
    }

    // 15. SCRUBBED SCENE UPDATE FUNCTION
    function updateProgress(p: number, sceneIdx: number) {
      // -----------------------------------------------------------------------
      // MONOLITH ARMOR GATES: Slide open between p = 0.00 and 0.25
      // -----------------------------------------------------------------------
      const gateOpenProgress = Math.min(1.0, Math.max(0, (p - 0.02) / 0.20));
      leftPlate.position.x = -4.3 - gateOpenProgress * 9.0;
      rightPlate.position.x = 4.3 + gateOpenProgress * 9.0;

      // -----------------------------------------------------------------------
      // SCENE 01: MONOLITH HULL BREACH & CONNECTED BASELINE (p < 0.18)
      // -----------------------------------------------------------------------
      if (p < 0.18) {
        shakeIntensity = 0;
        emergencyStrobeLight.intensity = 0;
        ambientLight.color.setHex(0x0f172a);
        ambientLight.intensity = 1.2;
        reactorLight.color.setHex(0x10b981);
        reactorLight.intensity = 4.5;
        targetLight.color.setHex(0x10b981);
        targetLight.intensity = 3.2;

        if (targetSpireObj) {
          targetSpireObj.beaconMat.color.setHex(0x10b981);
          targetSpireObj.haloMat.color.setHex(0x10b981);
          targetSpireObj.haloMat.opacity = 0.5;
        }

        if (targetWaveguide) {
          targetWaveguide.lineMat.color.setHex(0x10b981);
          targetWaveguide.lineMat.opacity = 0.9;
        }

        shockwaveMat1.opacity = 0;
        shockwaveMat2.opacity = 0;
        sparksMat.opacity = 0;
        recoveryWaveMat.opacity = 0;
      }

      // -----------------------------------------------------------------------
      // SCENE 02: ACTIVE TELEMETRY & EDGE PROBING (0.18 <= p < 0.38)
      // -----------------------------------------------------------------------
      else if (p >= 0.18 && p < 0.38) {
        const subP = (p - 0.18) / 0.20;
        shakeIntensity = 0;
        emergencyStrobeLight.intensity = 0;
        ambientLight.color.setHex(0x0a1628);
        ambientLight.intensity = 1.3;
        targetLight.color.setHex(0x06b6d4);
        targetLight.intensity = 3.5 + Math.sin(subP * Math.PI) * 1.5;

        if (targetSpireObj) {
          targetSpireObj.beaconMat.color.setHex(0x06b6d4);
          targetSpireObj.haloMat.color.setHex(0x06b6d4);
          targetSpireObj.haloMat.opacity = 0.65;
        }

        if (targetWaveguide) {
          targetWaveguide.lineMat.color.setHex(0x06b6d4);
          targetWaveguide.lineMat.opacity = 0.95;
        }

        shockwaveMat1.opacity = 0;
        shockwaveMat2.opacity = 0;
        sparksMat.opacity = 0;
        recoveryWaveMat.opacity = 0;
      }

      // -----------------------------------------------------------------------
      // SCENE 03: DEGRADATION & LATENCY ANOMALY (0.38 <= p < 0.50)
      // -----------------------------------------------------------------------
      else if (p >= 0.38 && p < 0.50) {
        const subP = (p - 0.38) / 0.12;
        shakeIntensity = subP * 0.05;
        emergencyStrobeLight.intensity = 0;
        ambientLight.color.setHex(0x241405);
        ambientLight.intensity = 1.2;
        targetLight.color.setHex(0xf59e0b);
        targetLight.intensity = 4.5 + Math.sin(subP * 15) * 2.0;

        if (targetSpireObj) {
          targetSpireObj.beaconMat.color.setHex(0xf59e0b);
          targetSpireObj.haloMat.color.setHex(0xf59e0b);
          targetSpireObj.haloMat.opacity = 0.75 + Math.sin(subP * 18) * 0.2;
        }

        if (targetWaveguide) {
          targetWaveguide.lineMat.color.setHex(0xf59e0b);
          targetWaveguide.lineMat.opacity = 1.0;

          // Vertex Jitter on target waveguide
          const posAttr = targetWaveguide.lineMesh.geometry.attributes.position;
          const posArr = posAttr.array as Float32Array;
          for (let i = 0; i < 51; i++) {
            const defPt = targetWaveguide.defaultPoints[i];
            const jitter = Math.sin(i * 1.2 + subP * 30) * subP * 0.32;
            posArr[i * 3] = defPt.x + jitter;
            posArr[i * 3 + 1] = defPt.y + (Math.cos(i + subP * 25) * subP * 0.25);
            posArr[i * 3 + 2] = defPt.z + jitter * 0.6;
          }
          posAttr.needsUpdate = true;
        }

        shockwaveMat1.opacity = 0;
        shockwaveMat2.opacity = 0;
        sparksMat.opacity = 0;
        recoveryWaveMat.opacity = 0;
      }

      // -----------------------------------------------------------------------
      // SCENE 04: CATASTROPHIC RUPTURE & TOTAL OUTAGE (0.50 <= p < 0.68)
      // Unexpected blackout at p=0.50, then violent crimson strobe & arc blast
      // -----------------------------------------------------------------------
      else if (p >= 0.50 && p < 0.68) {
        const subP = (p - 0.50) / 0.18;

        // Blackout moment between 0.50 and 0.52
        if (subP < 0.12) {
          ambientLight.intensity = 0.05;
          reactorLight.intensity = 0.2;
          targetLight.intensity = 0.1;
          emergencyStrobeLight.intensity = 0;
          shakeIntensity = 0.02;
        } else {
          // Violent emergency strobe
          shakeIntensity = Math.max(0, 0.24 - subP * 0.15);
          ambientLight.color.setHex(0x380606);
          ambientLight.intensity = 1.5;
          emergencyStrobeLight.intensity = 12.0 + Math.sin(subP * 25) * 4.0;
          targetLight.color.setHex(0xef4444);
          targetLight.intensity = 8.0 + Math.sin(subP * 20) * 4.0;
        }

        if (targetSpireObj) {
          targetSpireObj.beaconMat.color.setHex(0xef4444);
          targetSpireObj.haloMat.color.setHex(0xff2222);
          targetSpireObj.haloMat.opacity = 0.95;
        }

        // Physically sever waveguide into two disjoint recoiling stubs
        if (targetWaveguide) {
          const posAttr = targetWaveguide.lineMesh.geometry.attributes.position;
          const posArr = posAttr.array as Float32Array;
          const midIdx = 25;
          for (let i = 0; i < 51; i++) {
            const defPt = targetWaveguide.defaultPoints[i];
            if (i >= midIdx - 9 && i <= midIdx + 9) {
              posArr[i * 3] = NaN;
              posArr[i * 3 + 1] = NaN;
              posArr[i * 3 + 2] = NaN;
            } else {
              posArr[i * 3] = defPt.x;
              posArr[i * 3 + 1] = defPt.y;
              posArr[i * 3 + 2] = defPt.z;
            }
          }
          posAttr.needsUpdate = true;
          targetWaveguide.lineMat.color.setHex(0xef4444);
          targetWaveguide.lineMat.opacity = 0.8;
        }

        // Expand Shockwave
        const shockScale = 0.5 + subP * 18.0;
        shockwaveGroup.scale.set(shockScale, shockScale, shockScale);
        shockwaveMesh1.lookAt(camera.position);
        shockwaveMesh2.lookAt(camera.position);
        shockwaveMat1.opacity = Math.max(0, 0.95 - subP * 0.95);
        shockwaveMat2.opacity = Math.max(0, 0.75 - subP * 0.75);

        // Disperse Sparks
        sparksMat.opacity = Math.max(0, 1.0 - subP * 0.85);
        const sparksPosAttr = sparksGeo.attributes.position;
        const sparksPosArr = sparksPosAttr.array as Float32Array;
        for (let i = 0; i < sparksCount; i++) {
          sparksPosArr[i * 3] = sparksOrigin.x + sparksVelocities[i * 3] * subP * 4.2;
          sparksPosArr[i * 3 + 1] = sparksOrigin.y + sparksVelocities[i * 3 + 1] * subP * 4.2;
          sparksPosArr[i * 3 + 2] = sparksOrigin.z + sparksVelocities[i * 3 + 2] * subP * 4.2;
        }
        sparksPosAttr.needsUpdate = true;

        recoveryWaveMat.opacity = 0;
      }

      // -----------------------------------------------------------------------
      // SCENE 05: AUTONOMOUS INCIDENT COMMAND & TRIAGE (0.68 <= p < 0.84)
      // -----------------------------------------------------------------------
      else if (p >= 0.68 && p < 0.84) {
        shakeIntensity = 0;
        emergencyStrobeLight.intensity = 0;
        ambientLight.color.setHex(0x1a1033);
        ambientLight.intensity = 1.3;
        targetLight.color.setHex(0x8b5cf6);
        targetLight.intensity = 4.5;

        if (targetSpireObj) {
          targetSpireObj.beaconMat.color.setHex(0x8b5cf6);
          targetSpireObj.haloMat.color.setHex(0x8b5cf6);
          targetSpireObj.haloMat.opacity = 0.8;
        }

        if (targetWaveguide) {
          targetWaveguide.lineMat.color.setHex(0x8b5cf6);
          targetWaveguide.lineMat.opacity = 0.5;
        }

        shockwaveMat1.opacity = 0;
        shockwaveMat2.opacity = 0;
        sparksMat.opacity = 0;
        recoveryWaveMat.opacity = 0;
      }

      // -----------------------------------------------------------------------
      // SCENE 06: RESTORATION, RE-KNITTING & EQUILIBRIUM (0.84 <= p <= 1.00)
      // -----------------------------------------------------------------------
      else {
        const subP = (p - 0.84) / 0.16;
        shakeIntensity = 0;
        emergencyStrobeLight.intensity = 0;
        ambientLight.color.setHex(0x06281e);
        ambientLight.intensity = 1.4;
        reactorLight.color.setHex(0x10b981);
        reactorLight.intensity = 5.0;
        targetLight.color.setHex(0x10b981);
        targetLight.intensity = 4.0;

        if (targetSpireObj) {
          targetSpireObj.beaconMat.color.setHex(0x10b981);
          targetSpireObj.haloMat.color.setHex(0x34d399);
          targetSpireObj.haloMat.opacity = 0.85;
        }

        // Reconnect and restore target waveguide completely
        if (targetWaveguide) {
          const posAttr = targetWaveguide.lineMesh.geometry.attributes.position;
          const posArr = posAttr.array as Float32Array;
          for (let i = 0; i < 51; i++) {
            const defPt = targetWaveguide.defaultPoints[i];
            posArr[i * 3] = defPt.x;
            posArr[i * 3 + 1] = defPt.y;
            posArr[i * 3 + 2] = defPt.z;
          }
          posAttr.needsUpdate = true;
          targetWaveguide.lineMat.color.setHex(0x10b981);
          targetWaveguide.lineMat.opacity = 0.95;
        }

        // Expanding restorative wave
        const recScale = 0.5 + subP * 16.0;
        recoveryWaveMesh.scale.set(recScale, recScale, recScale);
        recoveryWaveMesh.lookAt(camera.position);
        recoveryWaveMat.opacity = Math.max(0, 0.9 - subP * 0.9);

        shockwaveMat1.opacity = 0;
        shockwaveMat2.opacity = 0;
        sparksMat.opacity = 0;
      }

      // Update camera along 3D CatmullRom trajectory
      updateCameraSpline(p);
    }

    // 16. HIGH-PERFORMANCE ANIMATION LOOP
    function animate() {
      animId = requestAnimationFrame(animate);
      if (!isVisible) return;

      const time = clock.getElapsedTime();

      // Reactor ring rotations
      ring1.rotation.z += 0.007;
      ring2.rotation.y -= 0.006;
      plasmaCoreMesh.rotation.y += 0.005;
      plasmaCoreMesh.rotation.x -= 0.003;

      // Align spire halos with camera
      builtSpires.forEach((spire) => {
        spire.haloMesh.lookAt(camera.position);
      });

      // Animate packet movements along waveguides
      waveguides.forEach((conn) => {
        if (conn.spire.def.isTarget && progressRef.current >= 0.50 && progressRef.current < 0.84) {
          conn.packetMesh.visible = false;
        } else {
          conn.packetMesh.visible = true;
          conn.packetOffset = (conn.packetOffset + conn.packetSpeed) % 1.0;
          const pt = conn.curve.getPointAt(conn.packetOffset);
          conn.packetMesh.position.copy(pt);
        }
      });

      renderer.render(scene, camera);
    }

    // Initial render call
    updateProgress(progressRef.current, activeSceneRef.current);
    animate();

    // 17. RESIZE LISTENER
    const handleResize = () => {
      if (!container) return;
      const w = container.clientWidth || window.innerWidth;
      const h = container.clientHeight || window.innerHeight;
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
      renderer.setSize(w, h);
    };
    window.addEventListener("resize", handleResize);

    // 18. INTERSECTION OBSERVER FOR OFFSCREEN PAUSING
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((e) => {
          isVisible = e.isIntersecting;
        });
      },
      { threshold: 0.05 }
    );
    observer.observe(container);

    // 19. STRICT GPU CLEANUP DISPOSAL
    sceneElementsRef.current = {
      updateProgress,
      dispose: () => {
        cancelAnimationFrame(animId);
        window.removeEventListener("resize", handleResize);
        observer.disconnect();

        renderer.dispose();
        if (renderer.domElement.parentElement) {
          renderer.domElement.parentElement.removeChild(renderer.domElement);
        }

        gridHelper.geometry.dispose();
        (gridHelper.material as THREE.Material).dispose();
        runwayGeo.dispose();
        runwayMat.dispose();
        colGeo.dispose();
        colMat.dispose();
        colGlowMat.dispose();
        gatePlateGeo.dispose();
        gatePlateMat.dispose();
        gateBevelMat.dispose();
        cageGeo.dispose();
        cageMat.dispose();
        ring1.geometry.dispose();
        (ring1.material as THREE.Material).dispose();
        ring2.geometry.dispose();
        (ring2.material as THREE.Material).dispose();
        plasmaCoreGeo.dispose();
        plasmaCoreMat.dispose();
        sparksGeo.dispose();
        sparksMat.dispose();
        shockwaveMesh1.geometry.dispose();
        shockwaveMat1.dispose();
        shockwaveMesh2.geometry.dispose();
        shockwaveMat2.dispose();
        recoveryWaveMesh.geometry.dispose();
        recoveryWaveMat.dispose();

        builtSpires.forEach((s) => {
          s.pedestalMesh.geometry.dispose();
          (s.pedestalMesh.material as THREE.Material).dispose();
          s.beaconMesh.geometry.dispose();
          s.beaconMat.dispose();
          s.haloMesh.geometry.dispose();
          s.haloMat.dispose();
        });

        waveguides.forEach((w) => {
          w.lineMesh.geometry.dispose();
          w.lineMat.dispose();
          w.packetMesh.geometry.dispose();
          (w.packetMesh.material as THREE.Material).dispose();
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
