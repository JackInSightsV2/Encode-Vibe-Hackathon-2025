import React, { useRef, useEffect, useState, useMemo } from 'react';
import * as THREE from 'three';
import { SafetyEvent, EventType } from '../services/cockpitService';

// 3D Node types
export interface Node3D {
  id: string;
  type: 'trace' | 'span' | 'threat' | 'provider';
  position: THREE.Vector3;
  color: THREE.Color;
  size: number;
  data: any;
  connections: string[];
  animated?: boolean;
  pulseIntensity?: number;
}

// Visualization configuration
export interface VisualizationConfig {
  showConnections: boolean;
  showThreatHeatmap: boolean;
  enableAnimations: boolean;
  nodeScale: number;
  connectionOpacity: number;
  threatSensitivity: number;
}

interface ThreeDVisualizationProps {
  width?: number;
  height?: number;
  events?: SafetyEvent[];
  config?: Partial<VisualizationConfig>;
  onNodeClick?: (node: Node3D) => void;
  onNodeHover?: (node: Node3D | null) => void;
}

export const ThreeDVisualization: React.FC<ThreeDVisualizationProps> = ({
  width = 800,
  height = 600,
  events = [],
  config = {}
}) => {
  const mountRef = useRef<HTMLDivElement>(null);
  const sceneRef = useRef<THREE.Scene>();
  const rendererRef = useRef<THREE.WebGLRenderer>();
  const cameraRef = useRef<THREE.PerspectiveCamera>();
  const controlsRef = useRef<any>();
  const frameRef = useRef<number>();
  const [isInitialized, setIsInitialized] = useState(false);
  const [selectedNode] = useState<Node3D | null>(null);
  const [hoveredNode] = useState<Node3D | null>(null);

  // Merge with default config
  const visualConfig: VisualizationConfig = useMemo(() => ({
    showConnections: true,
    showThreatHeatmap: true,
    enableAnimations: true,
    nodeScale: 1.0,
    connectionOpacity: 0.3,
    threatSensitivity: 0.8,
    ...config
  }), [config]);

  // Convert events to 3D nodes
  const nodes3D = useMemo(() => {
    const nodeMap = new Map<string, Node3D>();
    const connections = new Map<string, Set<string>>();

    events.forEach(event => {
      const nodeId = event.trace_id || event.id;
      
      if (!nodeMap.has(nodeId)) {
        // Create new node
        const node: Node3D = {
          id: nodeId,
          type: getNodeType(event),
          position: generatePosition(nodeId, events.length),
          color: getNodeColor(event),
          size: getNodeSize(event),
          data: event,
          connections: [],
          animated: event.type === EventType.THREAT || event.type === EventType.ALERT,
          pulseIntensity: getThreatIntensity(event)
        };
        nodeMap.set(nodeId, node);
      }

      // Update node properties based on event
      const node = nodeMap.get(nodeId)!;
      updateNodeFromEvent(node, event);

      // Add connections for spans
      if (event.span_id && event.trace_id) {
        if (!connections.has(event.trace_id)) {
          connections.set(event.trace_id, new Set());
        }
        connections.get(event.trace_id)!.add(event.span_id);
      }
    });

    // Apply connections
    connections.forEach((spanIds, traceId) => {
      const traceNode = nodeMap.get(traceId);
      if (traceNode) {
        traceNode.connections = Array.from(spanIds).filter(id => nodeMap.has(id));
      }
    });

    return Array.from(nodeMap.values());
  }, [events]);

  // Initialize Three.js scene
  useEffect(() => {
    if (!mountRef.current || isInitialized) return;

    // Scene setup
    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x0a0a0a);
    sceneRef.current = scene;

    // Camera setup
    const camera = new THREE.PerspectiveCamera(75, width / height, 0.1, 1000);
    camera.position.set(0, 0, 50);
    cameraRef.current = camera;

    // Renderer setup
    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    renderer.setSize(width, height);
    renderer.shadowMap.enabled = true;
    renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    rendererRef.current = renderer;

    // Add lighting
    const ambientLight = new THREE.AmbientLight(0x404040, 0.6);
    scene.add(ambientLight);

    const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
    directionalLight.position.set(50, 50, 50);
    directionalLight.castShadow = true;
    scene.add(directionalLight);

    // Add controls (you'll need to install three/examples/jsm/controls/OrbitControls)
    try {
      const { OrbitControls } = require('three/examples/jsm/controls/OrbitControls');
      const controls = new OrbitControls(camera, renderer.domElement);
      controls.enableDamping = true;
      controls.dampingFactor = 0.05;
      controlsRef.current = controls;
    } catch (e) {
      console.warn('OrbitControls not available');
    }

    // Add to DOM
    mountRef.current.appendChild(renderer.domElement);

    // Add raycaster for mouse interaction
    const raycaster = new THREE.Raycaster();
    const mouse = new THREE.Vector2();

    const onMouseMove = (event: MouseEvent) => {
      const rect = renderer.domElement.getBoundingClientRect();
      mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
      mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

      raycaster.setFromCamera(mouse, camera);
      // Handle hover logic here
    };

    const onClick = (event: MouseEvent) => {
      const rect = renderer.domElement.getBoundingClientRect();
      mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
      mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

      raycaster.setFromCamera(mouse, camera);
      // Handle click logic here
    };

    renderer.domElement.addEventListener('mousemove', onMouseMove);
    renderer.domElement.addEventListener('click', onClick);

    setIsInitialized(true);

    // Cleanup
    return () => {
      if (frameRef.current) {
        cancelAnimationFrame(frameRef.current);
      }
      renderer.domElement.removeEventListener('mousemove', onMouseMove);
      renderer.domElement.removeEventListener('click', onClick);
      if (mountRef.current && renderer.domElement) {
        mountRef.current.removeChild(renderer.domElement);
      }
      renderer.dispose();
    };
  }, [width, height, isInitialized]);

  // Update visualization when nodes change
  useEffect(() => {
    if (!sceneRef.current || !isInitialized) return;

    // Clear existing objects
    const objectsToRemove = sceneRef.current.children.filter(
      (child: any) => child.userData.isVisualizationNode
    );
    objectsToRemove.forEach((obj: any) => sceneRef.current!.remove(obj));

    // Add new nodes
    nodes3D.forEach(node => {
      const geometry = createNodeGeometry(node);
      const material = createNodeMaterial(node);
      const mesh = new THREE.Mesh(geometry, material);
      
      mesh.position.copy(node.position);
      mesh.userData = { isVisualizationNode: true, node };
      mesh.castShadow = true;
      mesh.receiveShadow = true;

      sceneRef.current!.add(mesh);

      // Add connections if enabled
      if (visualConfig.showConnections && node.connections.length > 0) {
        node.connections.forEach(connectionId => {
          const targetNode = nodes3D.find(n => n.id === connectionId);
          if (targetNode) {
            const connection = createConnection(node.position, targetNode.position, visualConfig);
            connection.userData = { isVisualizationNode: true };
            sceneRef.current!.add(connection);
          }
        });
      }
    });

    // Add threat heatmap if enabled
    if (visualConfig.showThreatHeatmap) {
      const heatmap = createThreatHeatmap(nodes3D);
      if (heatmap) {
        heatmap.userData = { isVisualizationNode: true };
        sceneRef.current.add(heatmap);
      }
    }
  }, [nodes3D, visualConfig, isInitialized]);

  // Animation loop
  useEffect(() => {
    if (!rendererRef.current || !sceneRef.current || !cameraRef.current) return;

    const animate = () => {
      frameRef.current = requestAnimationFrame(animate);

      // Update controls
      if (controlsRef.current) {
        controlsRef.current.update();
      }

      // Animate nodes
      if (visualConfig.enableAnimations) {
        animateNodes(sceneRef.current!);
      }

      // Render
      rendererRef.current!.render(sceneRef.current!, cameraRef.current!);
    };

    animate();

    return () => {
      if (frameRef.current) {
        cancelAnimationFrame(frameRef.current);
      }
    };
  }, [visualConfig.enableAnimations, isInitialized]);

  return (
    <div className="relative">
      <div ref={mountRef} className="border border-gray-700 rounded-lg overflow-hidden" />
      
      {/* Controls overlay */}
      <div className="absolute top-4 right-4 bg-gray-800 bg-opacity-90 p-4 rounded-lg">
        <h3 className="text-white text-sm font-semibold mb-2">3D Controls</h3>
        <div className="space-y-2 text-xs text-gray-300">
          <div>🖱️ Left click + drag: Rotate</div>
          <div>🖱️ Right click + drag: Pan</div>
          <div>🖱️ Scroll: Zoom</div>
          <div>🎯 Click nodes for details</div>
        </div>
      </div>

      {/* Node info panel */}
      {(selectedNode || hoveredNode) && (
        <div className="absolute bottom-4 left-4 bg-gray-800 bg-opacity-90 p-4 rounded-lg max-w-xs">
          <NodeInfoPanel node={selectedNode || hoveredNode!} />
        </div>
      )}
    </div>
  );
};

// Helper functions

function getNodeType(event: SafetyEvent): Node3D['type'] {
  if (event.trace_id && !event.span_id) return 'trace';
  if (event.span_id) return 'span';
  if (event.type === EventType.THREAT || event.type === EventType.ALERT) return 'threat';
  if (event.type === EventType.PROVIDER) return 'provider';
  return 'trace';
}

function getNodeColor(event: SafetyEvent): THREE.Color {
  switch (event.type) {
    case EventType.THREAT:
    case EventType.ALERT:
      return new THREE.Color(0xff4444);
    case EventType.MODERATION:
      return new THREE.Color(0xff8800);
    case EventType.PROVIDER:
      return new THREE.Color(0x4444ff);
    case EventType.METRIC:
      return new THREE.Color(0x44ff44);
    default:
      return new THREE.Color(0x888888);
  }
}

function getNodeSize(event: SafetyEvent): number {
  const baseSize = 0.5;
  const severityMultiplier = event.severity === 'high' ? 2 : event.severity === 'medium' ? 1.5 : 1;
  const typeMultiplier = event.type === EventType.THREAT ? 1.5 : 1;
  
  return baseSize * severityMultiplier * typeMultiplier;
}

function getThreatIntensity(event: SafetyEvent): number {
  if (event.type === EventType.THREAT && event.data?.confidence) {
    return event.data.confidence;
  }
  if (event.severity === 'high') return 0.9;
  if (event.severity === 'medium') return 0.6;
  if (event.severity === 'low') return 0.3;
  return 0.1;
}

function generatePosition(nodeId: string, totalNodes: number): THREE.Vector3 {
  // Simple hash-based positioning
  const hash = nodeId.split('').reduce((a, b) => {
    a = ((a << 5) - a) + b.charCodeAt(0);
    return a & a;
  }, 0);
  
  const radius = Math.sqrt(totalNodes) * 3;
  const theta = (hash * 0.618033988749895) % (2 * Math.PI);
  const phi = Math.acos(1 - 2 * ((hash * 0.618033988749895) % 1));
  
  return new THREE.Vector3(
    radius * Math.sin(phi) * Math.cos(theta),
    radius * Math.sin(phi) * Math.sin(theta),
    radius * Math.cos(phi)
  );
}

function updateNodeFromEvent(node: Node3D, event: SafetyEvent): void {
  // Update node properties based on latest event
  if (event.type === EventType.THREAT && event.data?.confidence) {
    node.pulseIntensity = event.data.confidence;
    node.animated = true;
  }
  
  // Update size based on importance
  if (event.severity === 'high') {
    node.size = Math.max(node.size, 1.0);
  }
}

function createNodeGeometry(node: Node3D): THREE.BufferGeometry {
  switch (node.type) {
    case 'threat':
      return new THREE.OctahedronGeometry(node.size);
    case 'provider':
      return new THREE.BoxGeometry(node.size, node.size, node.size);
    case 'span':
      return new THREE.CylinderGeometry(node.size * 0.5, node.size * 0.5, node.size);
    default:
      return new THREE.SphereGeometry(node.size, 16, 16);
  }
}

function createNodeMaterial(node: Node3D): THREE.Material {
  const material = new THREE.MeshLambertMaterial({
    color: node.color,
    transparent: true,
    opacity: 0.8
  });

  if (node.animated) {
    // Create emissive material for animated nodes
    return new THREE.MeshLambertMaterial({
      color: node.color,
      emissive: node.color.clone().multiplyScalar(0.3),
      transparent: true,
      opacity: 0.9
    });
  }

  return material;
}

function createConnection(start: THREE.Vector3, end: THREE.Vector3, config: VisualizationConfig): THREE.Line {
  const geometry = new THREE.BufferGeometry().setFromPoints([start, end]);
  const material = new THREE.LineBasicMaterial({
    color: 0x666666,
    transparent: true,
    opacity: config.connectionOpacity
  });
  
  return new THREE.Line(geometry, material);
}

function createThreatHeatmap(nodes: Node3D[]): THREE.Mesh | null {
  const threatNodes = nodes.filter(n => n.type === 'threat' && n.pulseIntensity);
  
  if (threatNodes.length === 0) return null;

  // Create a simple heatmap plane
  const geometry = new THREE.PlaneGeometry(100, 100, 32, 32);
  const material = new THREE.MeshBasicMaterial({
    color: 0xff0000,
    transparent: true,
    opacity: 0.1,
    side: THREE.DoubleSide
  });

  const heatmap = new THREE.Mesh(geometry, material);
  heatmap.rotation.x = -Math.PI / 2;
  heatmap.position.y = -20;

  return heatmap;
}

function animateNodes(scene: THREE.Scene): void {
  const time = Date.now() * 0.001;
  
  scene.children.forEach((child: any) => {
    if (child.userData.isVisualizationNode && child.userData.node?.animated) {
      const node = child.userData.node as Node3D;
      const mesh = child as THREE.Mesh;
      
      // Pulse animation
      if (node.pulseIntensity) {
        const scale = 1 + Math.sin(time * 4) * 0.2 * node.pulseIntensity;
        mesh.scale.setScalar(scale);
        
        // Update emissive intensity
        if (mesh.material instanceof THREE.MeshLambertMaterial) {
          const intensity = 0.3 + Math.sin(time * 4) * 0.2 * node.pulseIntensity;
          mesh.material.emissive = node.color.clone().multiplyScalar(intensity);
        }
      }
    }
  });
}

// Node info panel component
const NodeInfoPanel: React.FC<{ node: Node3D }> = ({ node }) => {
  return (
    <div className="text-white text-sm">
      <div className="font-semibold mb-2">Node Details</div>
      <div className="space-y-1">
        <div><span className="text-gray-400">ID:</span> {node.id}</div>
        <div><span className="text-gray-400">Type:</span> {node.type}</div>
        <div><span className="text-gray-400">Size:</span> {node.size.toFixed(2)}</div>
        {node.data?.type && (
          <div><span className="text-gray-400">Event Type:</span> {node.data.type}</div>
        )}
        {node.data?.severity && (
          <div><span className="text-gray-400">Severity:</span> {node.data.severity}</div>
        )}
        {node.pulseIntensity && (
          <div><span className="text-gray-400">Threat Level:</span> {(node.pulseIntensity * 100).toFixed(0)}%</div>
        )}
        {node.connections.length > 0 && (
          <div><span className="text-gray-400">Connections:</span> {node.connections.length}</div>
        )}
      </div>
    </div>
  );
};

export default ThreeDVisualization;