class AncientWoodMonitor {
    constructor() {
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.controls = null;
        this.clock = new THREE.Clock();
        this.currentBuilding = '应县木塔';
        this.timberModel = null;
        this.sensorsGroup = null;
        this.riskVoxels = null;
        this.concentrationGroup = null;
        this.tunnelNetwork = null;
        this.birdRadar = null;
        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.selectedObject = null;

        this.showSensors = true;
        this.showRisk = true;
        this.showConcentration = false;
        this.showTunnel = true;
        this.showBirds = false;

        this.sensorData = {};
        this.riskZones = [];
        this.alerts = [];
        this.tunnelData = null;
        this.strengthData = [];
        this.particleFilterData = null;
        this.birdData = [];
        this.deterrentStatus = null;

        this.chart = null;
        this.chartType = 'acoustic';

        this.init();
    }

    init() {
        this.setupScene();
        this.setupLights();
        this.createBuilding();
        this.createSensors();
        this.createRiskVoxels();
        this.createTunnelNetwork();
        this.createBirdRadar();
        this.setupEventListeners();
        this.loadData();
        this.setupChart();
        this.animate();
    }

    setupScene() {
        const canvas = document.getElementById('three-canvas');
        const container = canvas.parentElement;

        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x0a0e27);
        this.scene.fog = new THREE.Fog(0x0a0e27, 50, 150);

        this.camera = new THREE.PerspectiveCamera(
            60,
            container.clientWidth / container.clientHeight,
            0.1,
            1000
        );
        this.camera.position.set(30, 25, 40);

        this.renderer = new THREE.WebGLRenderer({ 
            canvas: canvas,
            antialias: true 
        });
        this.renderer.setSize(container.clientWidth, container.clientHeight);
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;

        this.controls = new THREE.OrbitControls(this.camera, this.renderer.domElement);
        this.controls.enableDamping = true;
        this.controls.dampingFactor = 0.05;
        this.controls.minDistance = 10;
        this.controls.maxDistance = 100;
        this.controls.maxPolarAngle = Math.PI / 2 + 0.1;

        const gridHelper = new THREE.GridHelper(60, 30, 0x2a2f4a, 0x1a1f3a);
        this.scene.add(gridHelper);

        const groundGeometry = new THREE.PlaneGeometry(100, 100);
        const groundMaterial = new THREE.MeshStandardMaterial({ 
            color: 0x15182e,
            roughness: 0.9,
            metalness: 0.1
        });
        const ground = new THREE.Mesh(groundGeometry, groundMaterial);
        ground.rotation.x = -Math.PI / 2;
        ground.receiveShadow = true;
        this.scene.add(ground);
    }

    setupLights() {
        const ambientLight = new THREE.AmbientLight(0x404060, 0.5);
        this.scene.add(ambientLight);

        const mainLight = new THREE.DirectionalLight(0xffffff, 0.8);
        mainLight.position.set(20, 40, 30);
        mainLight.castShadow = true;
        mainLight.shadow.mapSize.width = 2048;
        mainLight.shadow.mapSize.height = 2048;
        mainLight.shadow.camera.near = 0.5;
        mainLight.shadow.camera.far = 100;
        mainLight.shadow.camera.left = -30;
        mainLight.shadow.camera.right = 30;
        mainLight.shadow.camera.top = 30;
        mainLight.shadow.camera.bottom = -30;
        this.scene.add(mainLight);

        const fillLight = new THREE.DirectionalLight(0x4a9eff, 0.3);
        fillLight.position.set(-20, 20, -20);
        this.scene.add(fillLight);

        const pointLight1 = new THREE.PointLight(0xff6b6b, 0.5, 50);
        pointLight1.position.set(0, 15, 0);
        this.scene.add(pointLight1);
    }

    createBuilding() {
        if (this.timberModel) {
            this.scene.remove(this.timberModel.getGroup());
            this.timberModel.dispose();
        }

        this.timberModel = new TimberModel(this.currentBuilding, {
            woodColor: 0x8B4513,
            woodDarkColor: 0x654321,
            roofColor: 0x2f4f4f
        });

        this.scene.add(this.timberModel.getGroup());
        this.hideLoading();
    }

    createYingxianPagoda() {
        const woodColor = 0x8B4513;
        const woodDark = 0x654321;
        const roofColor = 0x2f4f4f;

        const baseGeometry = new THREE.CylinderGeometry(12, 14, 2, 8);
        const baseMaterial = new THREE.MeshStandardMaterial({ 
            color: 0x4a4a4a,
            roughness: 0.8
        });
        const base = new THREE.Mesh(baseGeometry, baseMaterial);
        base.position.y = 1;
        base.castShadow = true;
        base.receiveShadow = true;
        base.userData = { name: '塔基', type: 'structure' };
        this.buildingGroup.add(base);

        let currentHeight = 3;
        const floorHeights = [8, 7, 6, 5, 4];
        const floorRadii = [10, 8.5, 7, 5.5, 4];

        for (let floor = 0; floor < 5; floor++) {
            const height = floorHeights[floor];
            const radius = floorRadii[floor];

            const bodyGeometry = new THREE.CylinderGeometry(radius * 0.85, radius, height, 8, 1, true);
            const bodyMaterial = new THREE.MeshStandardMaterial({
                color: woodColor,
                roughness: 0.7,
                side: THREE.DoubleSide
            });
            const body = new THREE.Mesh(bodyGeometry, bodyMaterial);
            body.position.y = currentHeight + height / 2;
            body.castShadow = true;
            body.receiveShadow = true;
            body.userData = { name: `第${floor + 1}层塔身`, type: 'structure', floor: floor + 1 };
            this.buildingGroup.add(body);

            for (let i = 0; i < 8; i++) {
                const angle = (i / 8) * Math.PI * 2;
                const pillarGeometry = new THREE.CylinderGeometry(0.3, 0.3, height, 8);
                const pillarMaterial = new THREE.MeshStandardMaterial({ 
                    color: woodDark,
                    roughness: 0.6
                });
                const pillar = new THREE.Mesh(pillarGeometry, pillarMaterial);
                pillar.position.set(
                    Math.cos(angle) * radius * 0.75,
                    currentHeight + height / 2,
                    Math.sin(angle) * radius * 0.75
                );
                pillar.castShadow = true;
                pillar.userData = { name: `第${floor + 1}层立柱${i + 1}`, type: 'pillar', floor: floor + 1 };
                this.buildingGroup.add(pillar);
            }

            const eaveRadius = radius + 1.5;
            const eaveGeometry = new THREE.CylinderGeometry(eaveRadius, radius * 0.9, 0.8, 8);
            const eaveMaterial = new THREE.MeshStandardMaterial({ 
                color: roofColor,
                roughness: 0.5
            });
            const eave = new THREE.Mesh(eaveGeometry, eaveMaterial);
            eave.position.y = currentHeight + height;
            eave.castShadow = true;
            eave.receiveShadow = true;
            eave.userData = { name: `第${floor + 1}层斗拱`, type: 'dougong', floor: floor + 1 };
            this.buildingGroup.add(eave);

            const dougongCount = 8;
            for (let i = 0; i < dougongCount; i++) {
                const angle = (i / dougongCount) * Math.PI * 2;
                const dgGeometry = new THREE.BoxGeometry(0.8, 0.6, 0.8);
                const dgMaterial = new THREE.MeshStandardMaterial({ 
                    color: woodColor,
                    roughness: 0.6
                });
                const dg = new THREE.Mesh(dgGeometry, dgMaterial);
                dg.position.set(
                    Math.cos(angle) * radius * 0.95,
                    currentHeight + height - 0.2,
                    Math.sin(angle) * radius * 0.95
                );
                dg.castShadow = true;
                dg.userData = { name: `第${floor + 1}层斗拱组件${i + 1}`, type: 'dougong', floor: floor + 1 };
                this.buildingGroup.add(dg);
            }

            currentHeight += height;
        }

        const topBaseGeometry = new THREE.CylinderGeometry(3, 4, 1.5, 8);
        const topBaseMaterial = new THREE.MeshStandardMaterial({ 
            color: woodDark,
            roughness: 0.6
        });
        const topBase = new THREE.Mesh(topBaseGeometry, topBaseMaterial);
        topBase.position.y = currentHeight + 0.75;
        topBase.castShadow = true;
        topBase.userData = { name: '塔刹基座', type: 'structure' };
        this.buildingGroup.add(topBase);

        const spireGeometry = new THREE.ConeGeometry(1.5, 8, 8);
        const spireMaterial = new THREE.MeshStandardMaterial({ 
            color: 0xd4af37,
            roughness: 0.3,
            metalness: 0.7
        });
        const spire = new THREE.Mesh(spireGeometry, spireMaterial);
        spire.position.y = currentHeight + 1.5 + 4;
        spire.castShadow = true;
        spire.userData = { name: '塔刹', type: 'spire' };
        this.buildingGroup.add(spire);

        const ringGeometry = new THREE.TorusGeometry(1.8, 0.15, 8, 16);
        const ringMaterial = new THREE.MeshStandardMaterial({ 
            color: 0xd4af37,
            roughness: 0.3,
            metalness: 0.7
        });
        const ring = new THREE.Mesh(ringGeometry, ringMaterial);
        ring.position.y = currentHeight + 3;
        ring.rotation.x = Math.PI / 2;
        ring.userData = { name: '相轮', type: 'decoration' };
        this.buildingGroup.add(ring);
    }

    createFoguangTemple() {
        const woodColor = 0x8B4513;
        const woodDark = 0x654321;
        const roofColor = 0x2f4f4f;

        const platformGeometry = new THREE.BoxGeometry(35, 2, 20);
        const platformMaterial = new THREE.MeshStandardMaterial({ 
            color: 0x4a4a4a,
            roughness: 0.8
        });
        const platform = new THREE.Mesh(platformGeometry, platformMaterial);
        platform.position.y = 1;
        platform.receiveShadow = true;
        platform.userData = { name: '大殿台基', type: 'structure' };
        this.buildingGroup.add(platform);

        const hallWidth = 28;
        const hallDepth = 16;
        const hallHeight = 8;

        const wallGeometry = new THREE.BoxGeometry(hallWidth, hallHeight, hallDepth);
        const wallMaterial = new THREE.MeshStandardMaterial({
            color: woodColor,
            roughness: 0.7,
            side: THREE.DoubleSide
        });
        const walls = new THREE.Mesh(wallGeometry, wallMaterial);
        walls.position.y = 2 + hallHeight / 2;
        walls.castShadow = true;
        walls.receiveShadow = true;
        walls.userData = { name: '东大殿主体', type: 'structure' };
        this.buildingGroup.add(walls);

        const pillarPositions = [
            [-12, 7], [-8, 7], [-4, 7], [0, 7], [4, 7], [8, 7], [12, 7],
            [-12, -7], [-8, -7], [-4, -7], [0, -7], [4, -7], [8, -7], [12, -7]
        ];

        pillarPositions.forEach((pos, i) => {
            const pillarGeometry = new THREE.CylinderGeometry(0.35, 0.35, hallHeight, 8);
            const pillarMaterial = new THREE.MeshStandardMaterial({ 
                color: woodDark,
                roughness: 0.6
            });
            const pillar = new THREE.Mesh(pillarGeometry, pillarMaterial);
            pillar.position.set(pos[0], 2 + hallHeight / 2, pos[1]);
            pillar.castShadow = true;
            pillar.userData = { name: `立柱${i + 1}`, type: 'pillar' };
            this.buildingGroup.add(pillar);
        });

        const roofWidth = hallWidth + 6;
        const roofDepth = hallDepth + 6;
        const roofGeometry = new THREE.ConeGeometry(
            Math.hypot(roofWidth / 2, roofDepth / 2),
            5,
            4
        );
        const roofMaterial = new THREE.MeshStandardMaterial({ 
            color: roofColor,
            roughness: 0.5
        });
        const roof = new THREE.Mesh(roofGeometry, roofMaterial);
        roof.position.y = 2 + hallHeight + 2.5;
        roof.rotation.y = Math.PI / 4;
        roof.castShadow = true;
        roof.userData = { name: '大殿屋顶', type: 'roof' };
        this.buildingGroup.add(roof);

        const dougongCount = 12;
        for (let i = 0; i < dougongCount; i++) {
            let x, z, rotY;
            if (i < 6) {
                x = -10 + i * 4;
                z = hallDepth / 2 + 0.5;
                rotY = 0;
            } else {
                x = -10 + (i - 6) * 4;
                z = -hallDepth / 2 - 0.5;
                rotY = Math.PI;
            }

            const dgGeometry = new THREE.BoxGeometry(1.2, 0.8, 0.6);
            const dgMaterial = new THREE.MeshStandardMaterial({ 
                color: woodColor,
                roughness: 0.6
            });
            const dg = new THREE.Mesh(dgGeometry, dgMaterial);
            dg.position.set(x, 2 + hallHeight - 0.5, z);
            dg.rotation.y = rotY;
            dg.castShadow = true;
            dg.userData = { name: `斗拱${i + 1}`, type: 'dougong' };
            this.buildingGroup.add(dg);
        }

        const sideHallGeometry = new THREE.BoxGeometry(8, 5, 10);
        const sideHallMaterial = new THREE.MeshStandardMaterial({ 
            color: woodColor,
            roughness: 0.7
        });
        
        const sideHall1 = new THREE.Mesh(sideHallGeometry, sideHallMaterial);
        sideHall1.position.set(-25, 4.5, 0);
        sideHall1.castShadow = true;
        sideHall1.userData = { name: '文殊殿', type: 'side_hall' };
        this.buildingGroup.add(sideHall1);

        const sideHall2 = new THREE.Mesh(sideHallGeometry, sideHallMaterial);
        sideHall2.position.set(25, 4.5, 0);
        sideHall2.castShadow = true;
        sideHall2.userData = { name: '普贤殿', type: 'side_hall' };
        this.buildingGroup.add(sideHall2);

        const gateGeometry = new THREE.BoxGeometry(10, 6, 8);
        const gateMaterial = new THREE.MeshStandardMaterial({ 
            color: woodColor,
            roughness: 0.7
        });
        const gate = new THREE.Mesh(gateGeometry, gateMaterial);
        gate.position.set(0, 4, -25);
        gate.castShadow = true;
        gate.userData = { name: '山门', type: 'gate' };
        this.buildingGroup.add(gate);
    }

    createSensors() {
        if (this.sensorsGroup) {
            this.scene.remove(this.sensorsGroup);
        }

        this.sensorsGroup = new THREE.Group();

        const acousticCount = this.currentBuilding === '应县木塔' ? 30 : 25;
        const moistureCount = this.currentBuilding === '应县木塔' ? 25 : 20;

        for (let i = 0; i < acousticCount; i++) {
            const sensor = this.createAcousticSensor(i);
            this.sensorsGroup.add(sensor);
        }

        for (let i = 0; i < moistureCount; i++) {
            const sensor = this.createMoistureSensor(i);
            this.sensorsGroup.add(sensor);
        }

        this.sensorsGroup.visible = this.showSensors;
        this.scene.add(this.sensorsGroup);
    }

    createAcousticSensor(index) {
        const geometry = new THREE.SphereGeometry(0.25, 16, 16);
        const material = new THREE.MeshStandardMaterial({ 
            color: 0x4a9eff,
            emissive: 0x1a4e8a,
            emissiveIntensity: 0.5,
            roughness: 0.3,
            metalness: 0.6
        });
        const sensor = new THREE.Mesh(geometry, material);

        let x, y, z;
        if (this.currentBuilding === '应县木塔') {
            const floors = 5;
            const perFloor = Math.ceil(30 / floors);
            const floor = Math.floor(index / perFloor);
            const idxInFloor = index % perFloor;
            const angle = (idxInFloor / perFloor) * Math.PI * 2;
            const radius = 6 - floor * 0.8;
            
            x = Math.cos(angle) * radius;
            z = Math.sin(angle) * radius;
            y = 5 + floor * 6;
        } else {
            const angle = (index / 25) * Math.PI * 2;
            const radius = 10 + (index % 3) * 3;
            x = Math.cos(angle) * radius;
            z = Math.sin(angle) * radius * 0.6;
            y = 3 + (index % 5) * 1.2;
        }

        sensor.position.set(x, y, z);
        sensor.castShadow = true;
        
        sensor.userData = {
            name: `声发射传感器 ${index + 1}`,
            type: 'acoustic',
            sensorId: `AC-${this.currentBuilding === '应县木塔' ? 'YMT' : 'FGS'}-${String(index + 1).padStart(3, '0')}`,
            pos_x: x,
            pos_y: z,
            pos_z: y
        };

        const glowGeometry = new THREE.SphereGeometry(0.35, 16, 16);
        const glowMaterial = new THREE.MeshBasicMaterial({
            color: 0x4a9eff,
            transparent: true,
            opacity: 0.3
        });
        const glow = new THREE.Mesh(glowGeometry, glowMaterial);
        sensor.add(glow);

        return sensor;
    }

    createMoistureSensor(index) {
        const geometry = new THREE.CylinderGeometry(0.15, 0.15, 0.5, 8);
        const material = new THREE.MeshStandardMaterial({ 
            color: 0x2ed573,
            emissive: 0x0a5a2a,
            emissiveIntensity: 0.4,
            roughness: 0.4,
            metalness: 0.5
        });
        const sensor = new THREE.Mesh(geometry, material);

        let x, y, z;
        if (this.currentBuilding === '应县木塔') {
            const floors = 5;
            const perFloor = Math.ceil(25 / floors);
            const floor = Math.floor(index / perFloor);
            const idxInFloor = index % perFloor;
            const angle = (idxInFloor / perFloor) * Math.PI * 2 + 0.3;
            const radius = 5 - floor * 0.7;
            
            x = Math.cos(angle) * radius;
            z = Math.sin(angle) * radius;
            y = 4 + floor * 6;
        } else {
            x = -10 + (index % 8) * 3;
            z = -5 + Math.floor(index / 8) * 4;
            y = 2.5 + (index % 3) * 1.5;
        }

        sensor.position.set(x, y, z);
        sensor.castShadow = true;
        
        sensor.userData = {
            name: `含水率传感器 ${index + 1}`,
            type: 'moisture',
            sensorId: `MS-${this.currentBuilding === '应县木塔' ? 'YMT' : 'FGS'}-${String(index + 1).padStart(3, '0')}`,
            pos_x: x,
            pos_y: z,
            pos_z: y
        };

        return sensor;
    }

    createRiskVoxels() {
        if (this.riskVoxels) {
            this.riskVoxels.dispose();
        }

        this.riskVoxels = new VoxelRisk(this.scene, {
            defaultSize: 1.5,
            defaultOpacity: 0.4,
            criticalColor: 0xff4757,
            highColor: 0xffa500,
            mediumColor: 0xffeb00,
            lowColor: 0x4dc866,
            normalColor: 0x4a88ff,
            pulseEnabled: true,
            pulseSpeed: 0.02
        });

        this.riskVoxels.setVisible(this.showRisk);

        const numVoxels = 15;
        const mockZones = [];
        for (let i = 0; i < numVoxels; i++) {
            const intensity = Math.random();
            let x, y, z;
            if (this.currentBuilding === '应县木塔') {
                const angle = Math.random() * Math.PI * 2;
                const radius = 2 + Math.random() * 6;
                x = Math.cos(angle) * radius;
                z = Math.sin(angle) * radius;
                y = 3 + Math.random() * 25;
            } else {
                x = -12 + Math.random() * 24;
                z = -6 + Math.random() * 12;
                y = 2 + Math.random() * 8;
            }
            mockZones.push({
                sensor_id: `V-${i}`,
                pos_x: x,
                pos_y: z,
                pos_z: y,
                radius: 0.8 + intensity * 0.8,
                intensity: intensity,
                risk_level: intensity > 0.7 ? 'critical' : (intensity > 0.4 ? 'high' : 'medium'),
                event_rate: Math.floor(intensity * 120),
                location: `区域 ${i + 1}`
            });
        }
        this.riskZones = mockZones;
        this.riskVoxels.updateRiskZones(this.riskZones);
    }

    createTunnelNetwork() {
        if (this.tunnelNetwork) {
            this.tunnelNetwork.dispose();
        }
        this.tunnelNetwork = new TunnelNetwork(this.scene, {
            nodeSize: 0.3,
            activeColor: 0xff6b6b,
            inactiveColor: 0x666666,
            edgeColor: 0xff9944
        });
        this.tunnelNetwork.setVisible(this.showTunnel);
    }

    createBirdRadar() {
        if (this.birdRadar) {
            this.birdRadar.dispose();
        }
        this.birdRadar = new BirdRadarOverlay(this.scene, {
            scanRadius: 50
        });
        this.birdRadar.setVisible(this.showBirds);
    }

    _getRiskColor(riskLevel) {
        switch (riskLevel) {
            case 'critical': return new THREE.Color(1, 0.28, 0.34);
            case 'high': return new THREE.Color(1, 0.65, 0);
            case 'medium': return new THREE.Color(1, 0.92, 0);
            case 'low': return new THREE.Color(0.3, 0.8, 0.4);
            default: return new THREE.Color(0.3, 0.6, 1);
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

    updateRiskVoxels() {
        if (!this.riskZones || this.riskZones.length === 0) return;
        if (!this.riskVoxels) return;

        this.riskVoxels.updateRiskZones(this.riskZones);
    }

    createConcentrationField() {
        if (this.concentrationGroup) {
            this.scene.remove(this.concentrationGroup);
        }

        this.concentrationGroup = new THREE.Group();

        const gridSize = 15;
        const cellSize = 2;
        const centerX = 0;
        const centerY = 10;
        const centerZ = 0;

        for (let i = 0; i < gridSize; i++) {
            for (let j = 0; j < gridSize; j++) {
                for (let k = 0; k < gridSize; k++) {
                    const dx = (i - gridSize / 2) * cellSize;
                    const dy = (j - gridSize / 2) * cellSize;
                    const dz = (k - gridSize / 2) * cellSize;

                    const distance = Math.sqrt(dx * dx + dy * dy + dz * dz);
                    const concentration = Math.exp(-distance / 15) * (1 + 0.3 * Math.sin(dx * 0.2));

                    if (concentration < 0.05) continue;

                    const geometry = new THREE.BoxGeometry(cellSize * 0.9, cellSize * 0.9, cellSize * 0.9);
                    
                    const hue = 0.3 - concentration * 0.3;
                    const color = new THREE.Color().setHSL(hue, 1, 0.5);

                    const material = new THREE.MeshBasicMaterial({
                        color: color,
                        transparent: true,
                        opacity: concentration * 0.5,
                        depthWrite: false
                    });

                    const voxel = new THREE.Mesh(geometry, material);
                    voxel.position.set(
                        centerX + dx,
                        centerY + dy,
                        centerZ + dz
                    );
                    voxel.userData = {
                        type: 'concentration',
                        concentration: concentration
                    };

                    this.concentrationGroup.add(voxel);
                }
            }
        }

        this.concentrationGroup.visible = this.showConcentration;
        this.scene.add(this.concentrationGroup);
    }

    updateConcentrationFromResult(result) {
        if (!result || !result.points || result.points.length === 0) {
            return;
        }

        if (this.concentrationGroup) {
            this.scene.remove(this.concentrationGroup);
        }

        this.concentrationGroup = new THREE.Group();

        const maxConc = result.max_concentration || 1;

        result.points.forEach(point => {
            const concentration = point.concentration || 0;
            if (concentration < 0.001) return;

            const size = result.cell_size || 1;
            const geometry = new THREE.BoxGeometry(size * 0.85, size * 0.85, size * 0.85);
            
            const normalizedConc = Math.min(1, concentration / maxConc);
            const hue = 0.3 - normalizedConc * 0.3;
            const color = new THREE.Color().setHSL(hue, 1, 0.5);

            const material = new THREE.MeshBasicMaterial({
                color: color,
                transparent: true,
                opacity: normalizedConc * 0.6,
                depthWrite: false
            });

            const voxel = new THREE.Mesh(geometry, material);
            voxel.position.set(point.x || 0, point.z || 0, point.y || 0);
            voxel.userData = {
                type: 'concentration',
                concentration: concentration
            };

            this.concentrationGroup.add(voxel);
        });

        this.concentrationGroup.visible = this.showConcentration;
        this.scene.add(this.concentrationGroup);
    }

    setupEventListeners() {
        const canvas = this.renderer.domElement;
        
        canvas.addEventListener('click', (event) => this.onMouseClick(event));
        canvas.addEventListener('mousemove', (event) => this.onMouseMove(event));

        window.addEventListener('resize', () => this.onResize());

        document.querySelectorAll('.building-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.building-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.switchBuilding(btn.dataset.building);
            });
        });

        document.querySelectorAll('.view-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.view-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.setView(btn.dataset.view);
            });
        });

        document.getElementById('btn-show-sensors').addEventListener('click', (e) => {
            this.showSensors = !this.showSensors;
            e.target.classList.toggle('primary', this.showSensors);
            if (this.sensorsGroup) {
                this.sensorsGroup.visible = this.showSensors;
            }
        });

        document.getElementById('btn-show-risk').addEventListener('click', (e) => {
            this.showRisk = !this.showRisk;
            e.target.classList.toggle('primary', this.showRisk);
            if (this.riskVoxels) {
                this.riskVoxels.setVisible(this.showRisk);
            }
        });

        document.getElementById('btn-show-concentration').addEventListener('click', (e) => {
            this.showConcentration = !this.showConcentration;
            e.target.classList.toggle('primary', this.showConcentration);
            
            if (this.showConcentration && !this.concentrationGroup) {
                this.createConcentrationField();
            }
            if (this.concentrationGroup) {
                this.concentrationGroup.visible = this.showConcentration;
            }
            
            document.getElementById('concentration-legend').style.display = 
                this.showConcentration ? 'block' : 'none';
        });

        document.getElementById('btn-simulate').addEventListener('click', () => {
            document.getElementById('simulate-modal').classList.add('active');
        });

        document.getElementById('btn-release').addEventListener('click', () => {
            this.startFumigation();
        });

        document.getElementById('btn-show-tunnel').addEventListener('click', (e) => {
            this.showTunnel = !this.showTunnel;
            e.target.classList.toggle('primary', this.showTunnel);
            if (this.tunnelNetwork) {
                this.tunnelNetwork.setVisible(this.showTunnel);
            }
        });

        document.getElementById('btn-show-birds').addEventListener('click', (e) => {
            this.showBirds = !this.showBirds;
            e.target.classList.toggle('primary', this.showBirds);
            if (this.birdRadar) {
                this.birdRadar.setVisible(this.showBirds);
            }
        });

        document.getElementById('btn-bird-deterrent').addEventListener('click', () => {
            this.triggerBirdDeterrent();
        });

        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.chartType = btn.dataset.tab;
                this.updateChart();
            });
        });
    }

    onMouseClick(event) {
        const rect = this.renderer.domElement.getBoundingClientRect();
        this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

        this.raycaster.setFromCamera(this.mouse, this.camera);

        const objects = [];
        if (this.sensorsGroup && this.showSensors) {
            this.sensorsGroup.traverse(child => {
                if (child.isMesh && child.userData.sensorId) {
                    objects.push(child);
                }
            });
        }
        if (this.buildingGroup) {
            this.buildingGroup.traverse(child => {
                if (child.isMesh) {
                    objects.push(child);
                }
            });
        }
        if (this.riskVoxels && this.showRisk) {
            this.riskVoxels.getVoxels().forEach(voxel => {
                objects.push(voxel);
            });
        }

        const intersects = this.raycaster.intersectObjects(objects, false);

        if (intersects.length > 0) {
            const object = intersects[0].object;
            this.selectObject(object);
        } else {
            this.deselectObject();
        }
    }

    onMouseMove(event) {
        const rect = this.renderer.domElement.getBoundingClientRect();
        this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

        this.renderer.domElement.style.cursor = 'grab';
    }

    onResize() {
        const container = this.renderer.domElement.parentElement;
        this.camera.aspect = container.clientWidth / container.clientHeight;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(container.clientWidth, container.clientHeight);
    }

    selectObject(object) {
        if (this.selectedObject) {
            if (this.selectedObject.material.emissive) {
                this.selectedObject.material.emissive.setHex(this.selectedObject.userData.originalEmissive || 0x000000);
            }
        }

        this.selectedObject = object;

        if (object.material.emissive) {
            object.userData.originalEmissive = object.material.emissive.getHex();
            object.material.emissive.setHex(0xffff00);
        }

        this.updateInfoPanel(object);
    }

    deselectObject() {
        if (this.selectedObject && this.selectedObject.material.emissive) {
            this.selectedObject.material.emissive.setHex(this.selectedObject.userData.originalEmissive || 0x000000);
        }
        this.selectedObject = null;

        document.getElementById('info-name').textContent = '-';
        document.getElementById('info-location').textContent = '-';
        document.getElementById('info-events').textContent = '-';
        document.getElementById('info-moisture').textContent = '-';
        document.getElementById('info-risk').textContent = '-';
    }

    updateInfoPanel(object) {
        const data = object.userData;
        
        document.getElementById('info-name').textContent = data.name || '未知';
        document.getElementById('info-location').textContent = data.location || 
            `${data.pos_x?.toFixed(1) || 0}, ${data.pos_y?.toFixed(1) || 0}, ${data.pos_z?.toFixed(1) || 0}`;

        if (data.type === 'acoustic') {
            const eventRate = 20 + Math.random() * 80;
            document.getElementById('info-events').textContent = eventRate.toFixed(1) + ' 次/h';
            document.getElementById('info-moisture').textContent = '-';
            
            let risk = '低';
            if (eventRate > 100) risk = '严重';
            else if (eventRate > 70) risk = '高';
            else if (eventRate > 40) risk = '中';
            document.getElementById('info-risk').textContent = risk;
        } else if (data.type === 'moisture') {
            const moisture = 12 + Math.random() * 15;
            document.getElementById('info-events').textContent = '-';
            document.getElementById('info-moisture').textContent = moisture.toFixed(1) + '%';
            
            let risk = '正常';
            if (moisture > 25) risk = '偏高';
            else if (moisture > 20) risk = '略高';
            document.getElementById('info-risk').textContent = risk;
        } else if (data.type === 'risk_voxel') {
            document.getElementById('info-events').textContent = '-';
            document.getElementById('info-moisture').textContent = '-';
            
            const riskMap = { critical: '严重', high: '高', medium: '中', low: '低' };
            document.getElementById('info-risk').textContent = riskMap[data.riskLevel] || '未知';
        } else {
            document.getElementById('info-events').textContent = '-';
            document.getElementById('info-moisture').textContent = '-';
            document.getElementById('info-risk').textContent = '正常';
        }
    }

    switchBuilding(building) {
        this.currentBuilding = building;
        document.getElementById('building-title').textContent = `${building} - 三维监测模型`;
        
        this.showLoading();
        
        setTimeout(() => {
            this.createBuilding();
            this.createSensors();
            this.createRiskVoxels();
            this.createTunnelNetwork();
            this.createBirdRadar();
            this.loadData();
        }, 300);
    }

    setView(view) {
        const target = new THREE.Vector3(0, 15, 0);
        
        let position;
        switch (view) {
            case 'front':
                position = new THREE.Vector3(0, 15, 50);
                break;
            case 'side':
                position = new THREE.Vector3(50, 15, 0);
                break;
            case 'top':
                position = new THREE.Vector3(0, 60, 0.1);
                break;
            case 'perspective':
                position = new THREE.Vector3(30, 25, 40);
                break;
            case 'reset':
            default:
                position = new THREE.Vector3(30, 25, 40);
                break;
        }

        this.animateCamera(position, target);
    }

    animateCamera(targetPosition, targetLookAt) {
        const startPosition = this.camera.position.clone();
        const duration = 1000;
        const startTime = Date.now();

        const animate = () => {
            const elapsed = Date.now() - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            const easeProgress = 1 - Math.pow(1 - progress, 3);

            this.camera.position.lerpVectors(startPosition, targetPosition, easeProgress);
            this.controls.target.lerp(targetLookAt, easeProgress);
            this.controls.update();

            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };

        animate();
    }

    async apiRequest(path, options = {}) {
        const baseURL = '/api/v1';
        const url = baseURL + path;

        try {
            const response = await fetch(url, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            return await response.json();
        } catch (error) {
            console.warn(`API request failed for ${path}:`, error.message);
            return null;
        }
    }

    async loadData() {
        await Promise.all([
            this.loadAlerts(),
            this.loadRiskZones(),
            this.loadSensorStats(),
            this.loadTunnelNetwork(),
            this.loadStrengthAssessment(),
            this.loadFumigationTiming(),
            this.loadBirdRadar()
        ]);
        this.updateStats();
    }

    async loadAlerts() {
        const data = await this.apiRequest(`/alerts?building=${encodeURIComponent(this.currentBuilding)}`);
        
        if (data && data.alerts) {
            this.alerts = data.alerts.map(a => ({
                ...a,
                timestamp: new Date(a.timestamp)
            }));
        } else {
            this.alerts = [
                {
                    id: 'alert-1',
                    type: 'acoustic_emission',
                    severity: 'critical',
                    sensor_id: 'AC-YMT-012',
                    location: '三层斗拱东侧',
                    value: 145,
                    threshold: 100,
                    message: '声发射事件率过高，疑似严重白蚁活动',
                    timestamp: new Date(Date.now() - 1000 * 60 * 30)
                },
                {
                    id: 'alert-2',
                    type: 'wood_moisture',
                    severity: 'warning',
                    sensor_id: 'MS-YMT-008',
                    location: '二层北立柱',
                    value: 27.3,
                    threshold: 25,
                    message: '木材含水率偏高，存在虫蛀风险',
                    timestamp: new Date(Date.now() - 1000 * 60 * 60 * 2)
                },
                {
                    id: 'alert-3',
                    type: 'acoustic_emission',
                    severity: 'warning',
                    sensor_id: 'AC-YMT-025',
                    location: '五层南立柱',
                    value: 87,
                    threshold: 100,
                    message: '声发射事件率接近阈值，需密切关注',
                    timestamp: new Date(Date.now() - 1000 * 60 * 60 * 4)
                }
            ];
        }

        this.renderAlerts();
    }

    async loadRiskZones() {
        const data = await this.apiRequest(`/risk-zones?building=${encodeURIComponent(this.currentBuilding)}`);
        
        if (data && data.risk_zones) {
            this.riskZones = data.risk_zones;
            this.updateRiskVoxels();
        }
    }

    async loadSensorStats() {
        const data = await this.apiRequest(`/sensors?building=${encodeURIComponent(this.currentBuilding)}`);
        
        if (data && data.sensors) {
            const acoustic = data.sensors.filter(s => s.type === 'acoustic_emission');
            const moisture = data.sensors.filter(s => s.type === 'wood_moisture');
            const online = data.sensors.filter(s => s.status === 'online');
            
            document.getElementById('acoustic-count').textContent = acoustic.length + '台';
            document.getElementById('moisture-count').textContent = moisture.length + '台';
            document.getElementById('online-count').textContent = online.length + '台';
        }
    }

    async loadTunnelNetwork() {
        const data = await this.apiRequest(`/tdoa/tunnel-network?building=${encodeURIComponent(this.currentBuilding)}`);
        if (data && data.tunnel_network) {
            this.tunnelData = data.tunnel_network;
            if (this.tunnelNetwork) {
                this.tunnelNetwork.updateFromAPI(this.tunnelData);
            }
            const nodes = this.tunnelData.nodes || [];
            const activeNodes = nodes.filter(n => n.active);
            document.getElementById('tunnel-nodes').textContent = nodes.length;
            document.getElementById('tunnel-active').textContent = activeNodes.length;
            document.getElementById('tunnel-edges').textContent = (this.tunnelData.edges || []).length;
        }
    }

    async loadStrengthAssessment() {
        const data = await this.apiRequest(`/strength/assessment?building=${encodeURIComponent(this.currentBuilding)}`);
        if (data && data.assessments) {
            this.strengthData = data.assessments;
            const safe = this.strengthData.filter(a => a.strength_level === 'safe' || a.strength_level === 'caution').length;
            const danger = this.strengthData.filter(a => a.strength_level === 'danger' || a.strength_level === 'critical').length;
            const avgSF = this.strengthData.length > 0
                ? (this.strengthData.reduce((sum, a) => sum + a.safety_factor, 0) / this.strengthData.length).toFixed(2)
                : '-';
            document.getElementById('strength-safe').textContent = safe;
            document.getElementById('strength-danger').textContent = danger;
            document.getElementById('strength-avg-sf').textContent = avgSF;
        }
    }

    async loadFumigationTiming() {
        const data = await this.apiRequest(`/fumigation/timing?building=${encodeURIComponent(this.currentBuilding)}`);
        if (data && data.particle_filter) {
            this.particleFilterData = data.particle_filter;
            const pf = this.particleFilterData;
            if (pf.predicted_peak_time) {
                const peakTime = new Date(pf.predicted_peak_time);
                document.getElementById('pf-peak').textContent = peakTime.getHours() + ':' + String(peakTime.getMinutes()).padStart(2, '0');
            }
            if (pf.optimal_release_time) {
                const releaseTime = new Date(pf.optimal_release_time);
                document.getElementById('pf-release').textContent = releaseTime.getHours() + ':' + String(releaseTime.getMinutes()).padStart(2, '0');
                const el = document.getElementById('pf-release');
                if (pf.should_release_now) {
                    el.classList.add('critical');
                    el.classList.remove('warning', 'info');
                }
            }
            if (pf.confidence !== undefined) {
                document.getElementById('pf-confidence').textContent = (pf.confidence * 100).toFixed(0) + '%';
            }
        }
    }

    async loadBirdRadar() {
        const data = await this.apiRequest(`/bird/radar?building=${encodeURIComponent(this.currentBuilding)}`);
        if (data && data.scan_data) {
            this.birdData = data.scan_data;
            if (this.birdRadar) {
                this.birdRadar.updateBirds(this.birdData);
            }
            const woodpeckers = this.birdData.filter(b => b.bird_type === 'woodpecker').length;
            document.getElementById('bird-count').textContent = this.birdData.length;
            document.getElementById('bird-woodpecker').textContent = woodpeckers;
        }

        const statusData = await this.apiRequest(`/bird/deterrent/status?building=${encodeURIComponent(this.currentBuilding)}`);
        if (statusData) {
            this.deterrentStatus = statusData;
            const activeDeterrents = statusData.active_deterrents || [];
            const statusEl = document.getElementById('bird-deterrent-status');
            if (activeDeterrents.length > 0) {
                statusEl.textContent = '驱赶中';
                statusEl.classList.add('warning');
            } else {
                statusEl.textContent = '待机';
                statusEl.classList.remove('warning');
            }
        }
    }

    async triggerBirdDeterrent() {
        const data = await this.apiRequest('/bird/deterrent/trigger', {
            method: 'POST',
            body: JSON.stringify({
                building: this.currentBuilding,
                deterrent_type: 'ultrasonic'
            })
        });
        if (data) {
            if (this.birdRadar) {
                this.birdRadar.showDeterrentZone(this.currentBuilding, 'ultrasonic');
            }
            document.getElementById('bird-deterrent-status').textContent = '驱赶中';
            alert('超声波驱鸟装置已启动！');
        }
    }

    async simulateFumigation(params) {
        const data = await this.apiRequest('/simulate/fumigation', {
            method: 'POST',
            body: JSON.stringify(params)
        });
        
        return data;
    }

    renderAlerts() {
        const alertList = document.getElementById('alert-list');
        alertList.innerHTML = '';

        this.alerts.forEach(alert => {
            const severityClass = alert.severity === 'critical' ? '' : 
                                 alert.severity === 'warning' ? 'warning' : 'info';
            
            const alertItem = document.createElement('div');
            alertItem.className = `alert-item ${severityClass}`;
            alertItem.innerHTML = `
                <div class="alert-title">${alert.message}</div>
                <div class="alert-desc">${alert.sensor_id} · ${alert.location}</div>
                <div class="alert-time">${this.formatTime(alert.timestamp)}</div>
            `;
            alertList.appendChild(alertItem);
        });

        document.getElementById('alert-count').textContent = this.alerts.length;
    }

    formatTime(date) {
        const now = new Date();
        const diff = now - date;
        
        if (diff < 60000) {
            return '刚刚';
        } else if (diff < 3600000) {
            return Math.floor(diff / 60000) + ' 分钟前';
        } else if (diff < 86400000) {
            return Math.floor(diff / 3600000) + ' 小时前';
        } else {
            return Math.floor(diff / 86400000) + ' 天前';
        }
    }

    updateStats() {
        const acousticCount = this.currentBuilding === '应县木塔' ? 30 : 25;
        const moistureCount = this.currentBuilding === '应县木塔' ? 25 : 20;
        
        document.getElementById('acoustic-count').textContent = acousticCount + '台';
        document.getElementById('moisture-count').textContent = moistureCount + '台';
        document.getElementById('online-count').textContent = (acousticCount + moistureCount - 2) + '台';
    }

    setupChart() {
        const ctx = document.getElementById('data-chart').getContext('2d');
        
        this.chart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: '声发射事件率',
                    data: [],
                    borderColor: '#4a9eff',
                    backgroundColor: 'rgba(74, 158, 255, 0.1)',
                    fill: true,
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    x: {
                        grid: {
                            color: 'rgba(42, 47, 74, 0.5)'
                        },
                        ticks: {
                            color: '#8892b0',
                            font: { size: 10 }
                        }
                    },
                    y: {
                        grid: {
                            color: 'rgba(42, 47, 74, 0.5)'
                        },
                        ticks: {
                            color: '#8892b0',
                            font: { size: 10 }
                        }
                    }
                }
            }
        });

        this.updateChart();
    }

    updateChart() {
        if (!this.chart) return;

        const labels = [];
        const data = [];
        const now = new Date();

        for (let i = 23; i >= 0; i--) {
            const time = new Date(now - i * 60 * 60 * 1000);
            labels.push(time.getHours() + ':00');

            if (this.chartType === 'acoustic') {
                const base = 30 + Math.sin(i * 0.5) * 20;
                data.push(base + Math.random() * 15);
            } else if (this.chartType === 'moisture') {
                const base = 15 + Math.sin(i * 0.3) * 3;
                data.push(base + Math.random() * 2);
            } else {
                const base = 40 + Math.sin(i * 0.7) * 25;
                data.push(Math.max(0, base + Math.random() * 10));
            }
        }

        const labelMap = {
            acoustic: '声发射事件率 (次/h)',
            moisture: '含水率 (%)',
            prediction: '预测活动强度'
        };

        const colorMap = {
            acoustic: '#4a9eff',
            moisture: '#2ed573',
            prediction: '#ff6b81'
        };

        this.chart.data.labels = labels;
        this.chart.data.datasets[0].label = labelMap[this.chartType];
        this.chart.data.datasets[0].data = data;
        this.chart.data.datasets[0].borderColor = colorMap[this.chartType];
        this.chart.data.datasets[0].backgroundColor = colorMap[this.chartType] + '20';
        this.chart.update('none');
    }

    startFumigation() {
        if (confirm('确定要启动熏蒸剂释放吗？请确保现场人员已撤离。')) {
            this.showConcentration = true;
            document.getElementById('btn-show-concentration').classList.add('primary');
            
            if (!this.concentrationGroup) {
                this.createConcentrationField();
            }
            this.concentrationGroup.visible = true;
            document.getElementById('concentration-legend').style.display = 'block';

            alert('熏蒸剂释放已启动，预计持续 120 分钟。请通过浓度场监控扩散效果。');
        }
    }

    showLoading() {
        document.getElementById('loading').style.display = 'flex';
    }

    hideLoading() {
        document.getElementById('loading').style.display = 'none';
    }

    animate() {
        requestAnimationFrame(() => this.animate());

        const delta = this.clock ? this.clock.getDelta() : 0.016;
        const time = this.clock ? this.clock.elapsedTime : Date.now() * 0.001;

        if (this.riskVoxels && this.showRisk) {
            this.riskVoxels.update(delta);
        }

        if (this.tunnelNetwork && this.showTunnel) {
            this.tunnelNetwork.update(delta);
        }

        if (this.birdRadar && this.showBirds) {
            this.birdRadar.update(delta);
        }

        if (this.sensorsGroup && this.showSensors) {
            this.sensorsGroup.children.forEach((sensor, i) => {
                if (sensor.userData.type === 'acoustic') {
                    const pulse = 1 + Math.sin(time * 3 + i * 0.5) * 0.15;
                    sensor.scale.setScalar(pulse);
                }
            });
        }

        if (this.concentrationGroup && this.showConcentration) {
            this.concentrationGroup.children.forEach((voxel, i) => {
                const drift = Math.sin(time * 0.5 + i * 0.1) * 0.2;
                voxel.position.x += drift * 0.01;
            });
        }

        this.controls.update();
        this.renderer.render(this.scene, this.camera);
    }
}

function closeModal() {
    document.getElementById('simulate-modal').classList.remove('active');
}

async function runSimulation() {
    const releaseX = parseFloat(document.getElementById('release-x').value);
    const releaseY = parseFloat(document.getElementById('release-y').value);
    const releaseZ = parseFloat(document.getElementById('release-z').value);
    const releaseRate = parseFloat(document.getElementById('release-rate').value);
    const windSpeed = parseFloat(document.getElementById('wind-speed').value);
    const windDirection = parseFloat(document.getElementById('wind-direction').value);
    const duration = parseFloat(document.getElementById('duration').value);

    closeModal();

    const params = {
        building: app.currentBuilding,
        release_point_x: releaseX,
        release_point_y: releaseY,
        release_point_z: releaseZ,
        release_rate: releaseRate,
        wind_speed: windSpeed,
        wind_direction: windDirection,
        duration: duration
    };

    const result = await app.simulateFumigation(params);

    if (result && result.result) {
        app.updateConcentrationFromResult(result.result);
    }

    app.showConcentration = true;
    document.getElementById('btn-show-concentration').classList.add('primary');
    
    if (!app.concentrationGroup) {
        app.createConcentrationField();
    }
    app.concentrationGroup.visible = true;
    document.getElementById('concentration-legend').style.display = 'block';

    if (result && result.result) {
        alert(`熏蒸模拟完成：
最大浓度: ${result.result.max_concentration.toFixed(3)} g/m³
平均浓度: ${result.result.avg_concentration.toFixed(3)} g/m³
有效体积: ${result.result.effective_volume.toFixed(1)} m³

请查看浓度场可视化效果。`);
    } else {
        alert(`熏蒸模拟已启动（前端模拟模式）：
释放位置: (${releaseX}, ${releaseY}, ${releaseZ})
释放速率: ${releaseRate} g/s
风速: ${windSpeed} m/s
风向: ${windDirection}°
持续时间: ${duration} 分钟

请查看浓度场可视化效果。`);
    }
}

let app;
window.addEventListener('DOMContentLoaded', () => {
    app = new AncientWoodMonitor();
});
