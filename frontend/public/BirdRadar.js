class BirdRadarOverlay {
    constructor(scene, options = {}) {
        this.scene = scene;
        this.scanRadius = options.scanRadius || 50;
        this.visible = true;
        this.group = new THREE.Group();
        this.group.name = 'birdRadar';
        this.scene.add(this.group);
        this.currentBirds = [];
        this.radarRing = null;
        this.sweepAngle = 0;
        this.deterrentIndicators = [];
        this._createRadarBase();
    }

    _createRadarBase() {
        const ringGeometry = new THREE.RingGeometry(this.scanRadius - 0.5, this.scanRadius + 0.5, 64);
        const ringMaterial = new THREE.MeshBasicMaterial({
            color: 0x2ed573,
            transparent: true,
            opacity: 0.15,
            side: THREE.DoubleSide
        });
        this.radarRing = new THREE.Mesh(ringGeometry, ringMaterial);
        this.radarRing.rotation.x = -Math.PI / 2;
        this.radarRing.position.y = 35;
        this.group.add(this.radarRing);

        for (let i = 1; i <= 3; i++) {
            const innerRadius = (this.scanRadius / 4) * i - 0.2;
            const outerRadius = (this.scanRadius / 4) * i + 0.2;
            const gridRing = new THREE.Mesh(
                new THREE.RingGeometry(innerRadius, outerRadius, 64),
                new THREE.MeshBasicMaterial({ color: 0x2ed573, transparent: true, opacity: 0.08, side: THREE.DoubleSide })
            );
            gridRing.rotation.x = -Math.PI / 2;
            gridRing.position.y = 35;
            this.group.add(gridRing);
        }

        const sweepGeometry = new THREE.CircleGeometry(this.scanRadius, 64, 0, Math.PI / 6);
        const sweepMaterial = new THREE.MeshBasicMaterial({
            color: 0x2ed573,
            transparent: true,
            opacity: 0.08,
            side: THREE.DoubleSide
        });
        this.sweepMesh = new THREE.Mesh(sweepGeometry, sweepMaterial);
        this.sweepMesh.rotation.x = -Math.PI / 2;
        this.sweepMesh.position.y = 35;
        this.group.add(this.sweepMesh);
    }

    updateBirds(birdData) {
        this.currentBirds = birdData || [];
        this._clearBirdMarkers();

        this.currentBirds.forEach(bird => {
            const isWoodpecker = bird.bird_type === 'woodpecker';
            const color = isWoodpecker ? 0xff4757 : 0x2ed573;
            const size = isWoodpecker ? 0.6 : 0.35;

            const geometry = new THREE.ConeGeometry(size, size * 2, 6);
            const material = new THREE.MeshStandardMaterial({
                color: color,
                emissive: isWoodpecker ? 0x661122 : 0x0a3a1a,
                emissiveIntensity: 0.5,
                roughness: 0.4,
                metalness: 0.3
            });
            const marker = new THREE.Mesh(geometry, material);

            const dirRad = (bird.direction || 0) * Math.PI / 180;
            const dist = (bird.distance || 20) * (this.scanRadius / 100);
            const x = Math.cos(dirRad) * dist;
            const z = Math.sin(dirRad) * dist;
            const y = (bird.altitude || 10) * 0.8;

            marker.position.set(x, y + 30, z);
            marker.rotation.z = Math.PI;
            marker.userData = {
                type: 'bird_marker',
                birdType: bird.bird_type,
                birdId: bird.id,
                distance: bird.distance,
                altitude: bird.altitude,
                speed: bird.speed
            };

            this.group.add(marker);

            if (isWoodpecker) {
                const alertRing = new THREE.Mesh(
                    new THREE.RingGeometry(0.8, 1.2, 16),
                    new THREE.MeshBasicMaterial({ color: 0xff4757, transparent: true, opacity: 0.4, side: THREE.DoubleSide })
                );
                alertRing.rotation.x = -Math.PI / 2;
                alertRing.position.set(x, y + 30.1, z);
                this.group.add(alertRing);
            }
        });

        this.group.visible = this.visible;
    }

    showDeterrentZone(building, type) {
        const zoneRadius = 15;
        const color = type === 'ultrasonic' ? 0x4a9eff : 0xffa500;
        const geometry = new THREE.CylinderGeometry(zoneRadius, zoneRadius, 5, 32, 1, true);
        const material = new THREE.MeshBasicMaterial({
            color: color,
            transparent: true,
            opacity: 0.1,
            side: THREE.DoubleSide
        });
        const cylinder = new THREE.Mesh(geometry, material);
        cylinder.position.y = 20;
        cylinder.userData = { type: 'deterrent_zone', deterrentType: type };
        this.group.add(cylinder);
        this.deterrentIndicators.push({ mesh: cylinder, startTime: Date.now(), duration: 600000 });
    }

    _clearBirdMarkers() {
        const toRemove = [];
        this.group.children.forEach(child => {
            if (child.userData && (child.userData.type === 'bird_marker' || child.userData.type === 'deterrent_zone')) {
                toRemove.push(child);
            }
        });
        toRemove.forEach(child => {
            if (child.geometry) child.geometry.dispose();
            if (child.material) child.material.dispose();
            this.group.remove(child);
        });
    }

    setVisible(visible) {
        this.visible = visible;
        this.group.visible = visible;
    }

    update(delta) {
        if (!this.visible) return;
        this.sweepAngle += delta * 1.5;
        if (this.sweepAngle > Math.PI * 2) this.sweepAngle -= Math.PI * 2;
        if (this.sweepMesh) {
            this.sweepMesh.rotation.z = -this.sweepAngle;
        }

        const now = Date.now();
        this.deterrentIndicators = this.deterrentIndicators.filter(ind => {
            const elapsed = now - ind.startTime;
            if (elapsed > ind.duration) {
                this.group.remove(ind.mesh);
                if (ind.mesh.geometry) ind.mesh.geometry.dispose();
                if (ind.mesh.material) ind.mesh.material.dispose();
                return false;
            }
            ind.mesh.material.opacity = 0.1 * (1 - elapsed / ind.duration) * (0.5 + 0.5 * Math.sin(elapsed * 0.005));
            return true;
        });
    }

    dispose() {
        this._clearBirdMarkers();
        this.group.children.forEach(child => {
            if (child.geometry) child.geometry.dispose();
            if (child.material) child.material.dispose();
        });
        this.scene.remove(this.group);
    }
}
