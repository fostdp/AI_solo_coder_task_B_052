class TimberModel {
    constructor(buildingName, params = {}) {
        this.buildingName = buildingName;
        this.params = {
            woodColor: params.woodColor || 0x8B4513,
            woodDarkColor: params.woodDarkColor || 0x5C3A1D,
            woodLightColor: params.woodLightColor || 0xA0522D,
            roofColor: params.roofColor || 0x4A4A4A,
            ...params
        };

        this.group = new THREE.Group();
        this.build();
    }

    build() {
        if (this.buildingName === '应县木塔') {
            this.createPagoda();
        } else if (this.buildingName === '佛光寺') {
            this.createFoguangTemple();
        } else {
            this.createSimpleBuilding();
        }
    }

    createPagoda() {
        const baseGeometry = new THREE.CylinderGeometry(12, 14, 2, 8);
        const baseMaterial = new THREE.MeshLambertMaterial({
            color: this.params.woodDarkColor
        });
        const base = new THREE.Mesh(baseGeometry, baseMaterial);
        base.position.y = 1;
        base.castShadow = true;
        base.receiveShadow = true;
        this.group.add(base);

        const floors = 5;
        const floorHeight = 6;
        const roofOverhang = 2.5;

        for (let i = 0; i < floors; i++) {
            const scale = 1 - i * 0.12;
            const floorSize = 10 * scale;
            const y = 2 + i * floorHeight;

            const floorGeometry = new THREE.BoxGeometry(floorSize, floorHeight * 0.7, floorSize);
            const floorMaterial = new THREE.MeshLambertMaterial({
                color: this.params.woodColor
            });
            const floor = new THREE.Mesh(floorGeometry, floorMaterial);
            floor.position.y = y + floorHeight * 0.35;
            floor.castShadow = true;
            floor.receiveShadow = true;
            this.group.add(floor);

            const columns = 8;
            for (let j = 0; j < columns; j++) {
                const angle = (j / columns) * Math.PI * 2;
                const colRadius = floorSize * 0.4;
                const colGeometry = new THREE.CylinderGeometry(0.2, 0.25, floorHeight * 0.6, 6);
                const colMaterial = new THREE.MeshLambertMaterial({
                    color: this.params.woodLightColor
                });
                const column = new THREE.Mesh(colGeometry, colMaterial);
                column.position.set(
                    Math.cos(angle) * colRadius,
                    y + floorHeight * 0.5,
                    Math.sin(angle) * colRadius
                );
                column.castShadow = true;
                this.group.add(column);
            }

            const roofSize = floorSize + roofOverhang * scale;
            const roofGeometry = new THREE.ConeGeometry(roofSize, 1.5, 8);
            const roofMaterial = new THREE.MeshLambertMaterial({
                color: this.params.roofColor
            });
            const roof = new THREE.Mesh(roofGeometry, roofMaterial);
            roof.position.y = y + floorHeight * 0.8;
            roof.castShadow = true;
            this.group.add(roof);

            const eavesGeometry = new THREE.CylinderGeometry(
                roofSize * 0.9,
                roofSize,
                0.3,
                8
            );
            const eaves = new THREE.Mesh(eavesGeometry, roofMaterial);
            eaves.position.y = y + floorHeight * 0.7;
            eaves.castShadow = true;
            this.group.add(eaves);
        }

        const topSpireGeometry = new THREE.CylinderGeometry(0.3, 0.5, 4, 8);
        const topSpireMaterial = new THREE.MeshLambertMaterial({
            color: 0x2a2a2a
        });
        const topSpire = new THREE.Mesh(topSpireGeometry, topSpireMaterial);
        topSpire.position.y = 2 + floors * floorHeight + 1;
        topSpire.castShadow = true;
        this.group.add(topSpire);

        const beadGeometry = new THREE.SphereGeometry(0.6, 12, 8);
        const beadMaterial = new THREE.MeshLambertMaterial({
            color: 0xFFD700
        });
        const bead = new THREE.Mesh(beadGeometry, beadMaterial);
        bead.position.y = 2 + floors * floorHeight + 3.5;
        this.group.add(bead);
    }

    createFoguangTemple() {
        const platformGeometry = new THREE.BoxGeometry(30, 2, 18);
        const platformMaterial = new THREE.MeshLambertMaterial({
            color: 0x8B7355
        });
        const platform = new THREE.Mesh(platformGeometry, platformMaterial);
        platform.position.y = 1;
        platform.receiveShadow = true;
        this.group.add(platform);

        const hallGeometry = new THREE.BoxGeometry(24, 10, 14);
        const hallMaterial = new THREE.MeshLambertMaterial({
            color: this.params.woodColor
        });
        const hall = new THREE.Mesh(hallGeometry, hallMaterial);
        hall.position.y = 7;
        hall.castShadow = true;
        hall.receiveShadow = true;
        this.group.add(hall);

        const columns = [
            [-10, 7, -6], [-4, 7, -6], [4, 7, -6], [10, 7, -6],
            [-10, 7, 6], [-4, 7, 6], [4, 7, 6], [10, 7, 6]
        ];

        columns.forEach(([x, y, z]) => {
            const colGeometry = new THREE.CylinderGeometry(0.4, 0.5, 10, 8);
            const colMaterial = new THREE.MeshLambertMaterial({
                color: this.params.woodLightColor
            });
            const column = new THREE.Mesh(colGeometry, colMaterial);
            column.position.set(x, y, z);
            column.castShadow = true;
            this.group.add(column);
        });

        const roofGeometry = new THREE.BoxGeometry(28, 3, 18);
        const roofMaterial = new THREE.MeshLambertMaterial({
            color: this.params.roofColor
        });
        const roof = new THREE.Mesh(roofGeometry, roofMaterial);
        roof.position.y = 13.5;
        roof.castShadow = true;
        this.group.add(roof);

        const roofTopGeometry = new THREE.BoxGeometry(20, 1.5, 12);
        const roofTop = new THREE.Mesh(roofTopGeometry, roofMaterial);
        roofTop.position.y = 15;
        roofTop.castShadow = true;
        this.group.add(roofTop);

        const doorGeometry = new THREE.BoxGeometry(3, 4.5, 0.2);
        const doorMaterial = new THREE.MeshLambertMaterial({
            color: this.params.woodDarkColor
        });
        const door = new THREE.Mesh(doorGeometry, doorMaterial);
        door.position.set(0, 5, 7.05);
        this.group.add(door);

        const windowPositions = [
            [-7, 7, -7.05], [7, 7, -7.05],
            [-7, 7, 7.05], [7, 7, 7.05]
        ];
        const windowMaterial = new THREE.MeshLambertMaterial({
            color: 0x2a2a2a,
            transparent: true,
            opacity: 0.7
        });

        windowPositions.forEach(([x, y, z]) => {
            const winGeometry = new THREE.BoxGeometry(2, 2, 0.1);
            const window = new THREE.Mesh(winGeometry, windowMaterial);
            window.position.set(x, y, z);
            this.group.add(window);
        });

        const eastWing = new THREE.BoxGeometry(8, 6, 8);
        const eastWingMesh = new THREE.Mesh(eastWing, hallMaterial);
        eastWingMesh.position.set(-18, 5, 0);
        eastWingMesh.castShadow = true;
        this.group.add(eastWingMesh);

        const westWing = new THREE.BoxGeometry(8, 6, 8);
        const westWingMesh = new THREE.Mesh(westWing, hallMaterial);
        westWingMesh.position.set(18, 5, 0);
        westWingMesh.castShadow = true;
        this.group.add(westWingMesh);
    }

    createSimpleBuilding() {
        const geometry = new THREE.BoxGeometry(10, 8, 10);
        const material = new THREE.MeshLambertMaterial({
            color: this.params.woodColor
        });
        const building = new THREE.Mesh(geometry, material);
        building.position.y = 4;
        building.castShadow = true;
        building.receiveShadow = true;
        this.group.add(building);
    }

    getGroup() {
        return this.group;
    }

    setBuilding(buildingName) {
        this.buildingName = buildingName;
        this.group.clear();
        this.build();
    }

    dispose() {
        this.group.traverse((child) => {
            if (child.geometry) child.geometry.dispose();
            if (child.material) {
                if (Array.isArray(child.material)) {
                    child.material.forEach(m => m.dispose());
                } else {
                    child.material.dispose();
                }
            }
        });
    }
}

window.TimberModel = TimberModel;
