import { PhysicsEnvironment3D } from './PhysicsEnvironment3D';
import { PhysicsEntity3D } from './PhysicsEntity3D';

export default function PhysicsDemo() {
  const entityMetadata = {
    entity_id: 'ball_1',
    physics_type: 'rigid',
    radius: 1,
    color: 0xff5733,
    base_properties: {
      mass: 5.0,
      friction: 0.2,
      restitution: 0.8
    },
    material: {
      type: 'rubber',
      custom_properties: { bounciness: 0.9 }
    },
    dynamic_properties: {
      thermal: { conductivity: 0.3, temperature: 50.0 }
    }
  };

  return (
    <PhysicsEnvironment3D gravity={[0, -9.8, 0]} temperature={22} airDensity={1.2}>
      <PhysicsEntity3D entityId="ball_1" metadata={entityMetadata} />
    </PhysicsEnvironment3D>
  );
}
