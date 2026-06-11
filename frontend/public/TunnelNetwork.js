class TunnelNetwork {
    constructor(scene, options = {}) {
        this.scene = scene;
        this.nodeSize = options.nodeSize || 0.3;
        this.activeColor = options.activeColor || 0xff6b6b;
        this.inactiveColor = options.inactiveColor || 0x666666;
        this.edgeColor = options.edgeColor || 0xff9944;
        this.visible = true;
        this.group = new THREE.Group();
        this.group.name = 'tunnelNetwork';
        this.scene.add(this.group);
        this.currentData = null;
    }

    updateFromAPI(networkData) {
        if (!networkData) return;

        const dataHash = JSON.stringify(networkData);
        if (dataHash === this.currentData) return;
        this.currentData = dataHash;

        this.clear();

        if (!networkData.nodes || networkData.nodes.length === 0) return;

        const nodeMap = {};
        networkData.nodes.forEach(node => {
            const geometry = new THREE.SphereGeometry(this.nodeSize * (0.5 + node.confidence * 0.5), 16, 16);
            const material = new THREE.MeshStandardMaterial({
                color: node.active ? this.activeColor : this.inactiveColor,
                emissive: node.active ? 0x661111 : 0x222222,
                emissiveIntensity: node.active ? 0.6 : 0.2,
                roughness: 0.4,
                metalness: 0.3,
                transparent: true,
                opacity: node.active ? 0.9 : 0.4
            });
            const mesh = new THREE.Mesh(geometry, material);
            mesh.position.set(node.position_x, node.position_z, node.position_y);
            mesh.userData = {
                name: `蚁道节点 ${node.id}`,
                type: 'tunnel_node',
                nodeId: node.id,
                confidence: node.confidence,
                active: node.active
            };

            if (node.active) {
                const glowGeometry = new THREE.SphereGeometry(this.nodeSize * 1.5, 16, 16);
                const glowMaterial = new THREE.MeshBasicMaterial({
                    color: this.activeColor,
                    transparent: true,
                    opacity: 0.2
                });
                const glow = new THREE.Mesh(glowGeometry, glowMaterial);
                mesh.add(glow);
            }

            this.group.add(mesh);
            nodeMap[node.id] = mesh.position.clone();
        });

        if (networkData.edges) {
            networkData.edges.forEach(edge => {
                const fromPos = nodeMap[edge.from_node_id];
                const toPos = nodeMap[edge.to_node_id];
                if (!fromPos || !toPos) return;

                const points = [fromPos, toPos];
                const geometry = new THREE.BufferGeometry().setFromPoints(points);
                const material = new THREE.LineBasicMaterial({
                    color: this.edgeColor,
                    transparent: true,
                    opacity: 0.3 + edge.strength * 0.5,
                    linewidth: 1
                });
                const line = new THREE.Line(geometry, material);
                line.userData = {
                    type: 'tunnel_edge',
                    fromNode: edge.from_node_id,
                    toNode: edge.to_node_id,
                    strength: edge.strength,
                    length: edge.length
                };
                this.group.add(line);
            });
        }

        this.group.visible = this.visible;
    }

    clear() {
        while (this.group.children.length > 0) {
            const child = this.group.children[0];
            if (child.geometry) child.geometry.dispose();
            if (child.material) {
                if (Array.isArray(child.material)) {
                    child.material.forEach(m => m.dispose());
                } else {
                    child.material.dispose();
                }
            }
            child.traverse(sub => {
                if (sub.geometry) sub.geometry.dispose();
                if (sub.material) sub.material.dispose();
            });
            this.group.remove(child);
        }
        this.currentData = null;
    }

    setVisible(visible) {
        this.visible = visible;
        this.group.visible = visible;
    }

    dispose() {
        this.clear();
        this.scene.remove(this.group);
    }

    update(delta) {
        if (!this.visible) return;
        const time = Date.now() * 0.001;
        this.group.children.forEach((child, i) => {
            if (child.isMesh && child.userData.type === 'tunnel_node' && child.userData.active) {
                const pulse = 1 + Math.sin(time * 2 + i * 0.3) * 0.1;
                child.scale.setScalar(pulse);
            }
        });
    }

    getNodes() {
        return this.group.children.filter(c => c.isMesh && c.userData.type === 'tunnel_node');
    }
}
