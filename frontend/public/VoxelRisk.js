class VoxelRisk {
    constructor(scene, params = {}) {
        this.scene = scene;
        this.params = {
            defaultSize: params.defaultSize || 1.5,
            defaultOpacity: params.defaultOpacity || 0.4,
            criticalColor: params.criticalColor || 0xff4444,
            highColor: params.highColor || 0xffa500,
            mediumColor: params.mediumColor || 0xffeb00,
            lowColor: params.lowColor || 0x4dc866,
            normalColor: params.normalColor || 0x4a88ff,
            pulseEnabled: params.pulseEnabled !== undefined ? params.pulseEnabled : true,
            pulseSpeed: params.pulseSpeed || 0.02,
            pulseMin: params.pulseMin || 0.7,
            pulseMax: params.pulseMax || 1.3,
            ...params
        };

        this.group = new THREE.Group();
        this.scene.add(this.group);

        this.voxelMap = new Map();
        this.lastRiskHash = '';
        this.visible = true;
        this.time = 0;
    }

    getRiskColor(riskLevel) {
        switch (riskLevel) {
            case 'critical': return new THREE.Color(this.params.criticalColor);
            case 'high': return new THREE.Color(this.params.highColor);
            case 'medium': return new THREE.Color(this.params.mediumColor);
            case 'low': return new THREE.Color(this.params.lowColor);
            default: return new THREE.Color(this.params.normalColor);
        }
    }

    _createVoxelMesh(size, color, opacity) {
        const geometry = new THREE.BoxGeometry(size, size, size);
        const material = new THREE.MeshBasicMaterial({
            color: color,
            transparent: true,
            opacity: opacity,
            wireframe: false
        });
        const mesh = new THREE.Mesh(geometry, material);

        const edgesGeometry = new THREE.EdgesGeometry(geometry);
        const edgesMaterial = new THREE.LineBasicMaterial({
            color: color,
            transparent: true,
            opacity: 0.8
        });
        const edges = new THREE.LineSegments(edgesGeometry, edgesMaterial);
        mesh.add(edges);

        return mesh;
    }

    updateRiskZones(riskZones) {
        if (!riskZones || riskZones.length === 0) return;

        const newHash = JSON.stringify(riskZones.map(z => ({
            id: z.sensor_id,
            r: z.risk_level,
            i: z.intensity,
            x: z.pos_x,
            y: z.pos_y,
            z: z.pos_z
        })));

        if (newHash === this.lastRiskHash) return false;
        this.lastRiskHash = newHash;

        const currentKeys = new Set();

        riskZones.forEach(zone => {
            const key = zone.sensor_id || `zone-${zone.pos_x}-${zone.pos_y}-${zone.pos_z}`;
            currentKeys.add(key);

            const intensity = zone.intensity || 0.5;
            const size = (zone.radius || 1) * this.params.defaultSize;
            const riskLevel = zone.risk_level || 'medium';
            const color = this.getRiskColor(riskLevel);
            const opacity = this.params.defaultOpacity + intensity * 0.4;
            const posX = zone.pos_x || 0;
            const posY = zone.pos_z || 5;
            const posZ = zone.pos_y || 0;

            if (this.voxelMap.has(key)) {
                const existing = this.voxelMap.get(key);
                existing.material.color.copy(color);
                existing.material.opacity = opacity;
                existing.position.set(posX, posY, posZ);
                existing.scale.setScalar(1);
                existing.userData.intensity = intensity;
                existing.userData.riskLevel = riskLevel;
                existing.userData.eventRate = zone.event_rate;
                existing.userData.name = zone.location || '风险区域';

                const edgeMat = existing.children[0]?.material;
                if (edgeMat) edgeMat.color.copy(color);
            } else {
                const voxel = this._createVoxelMesh(size, color, opacity);
                voxel.position.set(posX, posY, posZ);
                voxel.userData = {
                    type: 'risk_voxel',
                    intensity: intensity,
                    riskLevel: riskLevel,
                    sensorId: zone.sensor_id,
                    eventRate: zone.event_rate,
                    name: zone.location || '风险区域',
                    baseScale: 1
                };
                this.group.add(voxel);
                this.voxelMap.set(key, voxel);
            }
        });

        for (const [key, voxel] of this.voxelMap) {
            if (!currentKeys.has(key)) {
                this.group.remove(voxel);
                if (voxel.geometry) voxel.geometry.dispose();
                if (voxel.material) voxel.material.dispose();
                const edges = voxel.children[0];
                if (edges) {
                    if (edges.geometry) edges.geometry.dispose();
                    if (edges.material) edges.material.dispose();
                }
                this.voxelMap.delete(key);
            }
        }

        return true;
    }

    update(deltaTime) {
        if (!this.params.pulseEnabled || !this.visible) return;

        this.time += deltaTime;

        for (const [key, voxel] of this.voxelMap) {
            const intensity = voxel.userData.intensity || 0.5;
            const riskLevel = voxel.userData.riskLevel || 'low';

            let pulseAmount = 0;
            if (riskLevel === 'critical') {
                pulseAmount = 0.3 * intensity;
            } else if (riskLevel === 'high') {
                pulseAmount = 0.15 * intensity;
            } else if (riskLevel === 'medium') {
                pulseAmount = 0.08 * intensity;
            }

            if (pulseAmount > 0) {
                const pulse = 1 + pulseAmount * Math.sin(this.time * this.params.pulseSpeed * 60);
                voxel.scale.setScalar(pulse);
            }
        }
    }

    setVisible(visible) {
        this.visible = visible;
        this.group.visible = visible;
    }

    toggle() {
        this.visible = !this.visible;
        this.group.visible = this.visible;
        return this.visible;
    }

    getVoxels() {
        return Array.from(this.voxelMap.values());
    }

    getCount() {
        return this.voxelMap.size;
    }

    clear() {
        for (const [key, voxel] of this.voxelMap) {
            this.group.remove(voxel);
            if (voxel.geometry) voxel.geometry.dispose();
            if (voxel.material) voxel.material.dispose();
        }
        this.voxelMap.clear();
        this.lastRiskHash = '';
    }

    dispose() {
        this.clear();
        this.scene.remove(this.group);
    }
}

window.VoxelRisk = VoxelRisk;
