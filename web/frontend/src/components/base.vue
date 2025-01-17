<script setup lang="ts">
import { grpc } from '@improbable-eng/grpc-web';
import { ref } from 'vue';
import { computeKeyboardBaseControls, BaseControlHelper } from '../rc/control_helpers';
import baseApi from '../gen/proto/api/component/base/v1/base_pb.esm';
import commonApi from '../gen/proto/api/common/v1/common_pb.esm';
import streamApi from '../gen/proto/stream/v1/stream_pb.esm';
import { toast } from '../lib/toast';
import { filterResources, type Resource } from '../lib/resource';
import { displayError } from '../lib/error';
import KeyboardInput from './keyboard-input.vue';

interface Props {
  name: string;
  resources: Resource[];
}

const props = defineProps<Props>();

type Tabs = 'Keyboard' | 'Discrete'
type MovementTypes = 'Continuous' | 'Discrete'
type MovementModes = 'Straight' | 'Spin'
type SpinTypes = 'Clockwise' | 'Counterclockwise'
type Directions = 'Forwards' | 'Backwards'

interface Emits {
  (event: 'showcamera', value: string): void
}

const emit = defineEmits<Emits>();

const selectedItem = ref<Tabs>('Keyboard');
const movementMode = ref<MovementModes>('Straight');
const movementType = ref<MovementTypes>('Continuous');
const direction = ref<Directions>('Forwards');
const spinType = ref<SpinTypes>('Clockwise');
const increment = ref(1000);
const speed = ref(200); // straight mm/s
const spinSpeed = ref(90); // spin deg/s
const angle = ref(0);

const handleTabSelect = (tab: Tabs) => {
  selectedItem.value = tab;

  if (tab === 'Keyboard') {
    viewPreviewCamera(props.name, true);
  } else {
    viewPreviewCamera(props.name, false);
    resetDiscreteState();
  }
};

const resetDiscreteState = () => {
  movementMode.value = 'Straight';
  movementType.value = 'Continuous';
  direction.value = 'Forwards';
  spinType.value = 'Clockwise';
};

const setMovementMode = (mode: MovementModes) => {
  movementMode.value = mode;
};

const setMovementType = (type: MovementTypes) => {
  movementType.value = type;
};

const setSpinType = (type: SpinTypes) => {
  spinType.value = type;
};

const setDirection = (dir: Directions) => {
  direction.value = dir;
};

const baseRun = () => {
  if (movementMode.value === 'Spin') {
    BaseControlHelper.spin(
      props.name,
      angle.value * (spinType.value === 'Clockwise' ? -1 : 1),
      spinSpeed.value,
      displayError
    );
  } else if (movementMode.value === 'Straight') {
    handleBaseStraight(props.name, {
      movementType: movementType.value,
      direction: direction.value === 'Forwards' ? 1 : -1,
      speed: speed.value,
      distance: increment.value,
    });
  } else {
    toast.error(`Unrecognized discrete movement mode: ${movementMode.value}`);
  }
};

const baseKeyboardCtl = (name: string, controls: Record<string, boolean>) => {
  if (Object.values(controls).every((item) => item === false)) {
    handleBaseActionStop(name);
    return;
  }

  const inputs = computeKeyboardBaseControls(controls);
  const linear = new commonApi.Vector3();
  const angular = new commonApi.Vector3();
  linear.setY(inputs.linear);
  angular.setZ(inputs.angular);
  BaseControlHelper.setPower(name, linear, angular, displayError);
};

const handleBaseActionStop = (name: string) => {
  const req = new baseApi.StopRequest();
  req.setName(name);
  window.baseService.stop(req, new grpc.Metadata(), displayError);
};

const handleBaseStraight = (name: string, event: {
  distance: number
  speed: number
  direction: number
  movementType: MovementTypes
}) => {
  if (event.movementType === 'Continuous') {
    const linear = new commonApi.Vector3();
    linear.setY(event.speed * event.direction);

    BaseControlHelper.setVelocity(
      name,
      linear, // linear
      new commonApi.Vector3(), // angular
      displayError
    );
  } else {
    BaseControlHelper.moveStraight(
      name,
      event.distance,
      event.speed * event.direction,
      displayError
    );
  }
};

const viewPreviewCamera = (name: string, isOn: boolean) => {
  if (isOn) {
    const req = new streamApi.AddStreamRequest();
    req.setName(name);
    window.streamService.addStream(req, new grpc.Metadata(), displayError);
    return;
  }

  const req = new streamApi.RemoveStreamRequest();
  req.setName(name);
  window.streamService.removeStream(req, new grpc.Metadata(), displayError);
};

const handleSelectCamera = (event: string) => {
  emit('showcamera', event);
};
</script>

<template>
  <v-collapse
    :title="name"
    class="base"
  >
    <v-breadcrumbs
      slot="title"
      crumbs="base"
    />

    <v-button
      slot="header"
      variant="danger"
      icon="stop-circle"
      label="STOP"
      @click="handleBaseActionStop(name)"
    />

    <div class="border border-t-0 border-black pt-2">
      <v-tabs
        tabs="Keyboard, Discrete"
        :selected="selectedItem"
        @input="handleTabSelect($event.detail.value)"
      />

      <div
        v-if="selectedItem === 'Keyboard'"
        class="h-auto p-4"
      >
        <div class="grid grid-cols-2">
          <div class="mt-2">
            <KeyboardInput @keyboard-ctl="baseKeyboardCtl(name, $event)" />
          </div>
          <div v-if="filterResources(resources, 'rdk', 'component', 'camera')">
            <v-select
              class="mb-4"
              variant="multiple"
              placeholder="Select Cameras"
              :options="
                filterResources(resources, 'rdk', 'component', 'camera')
                  .map(({ name }) => name)
                  .join(',')
              "
              @input="handleSelectCamera($event.detail.value)"
            />
            <template
              v-for="basecamera in filterResources(
                resources,
                'rdk',
                'component',
                'camera'
              )"
              :key="basecamera.name"
            >
              <div
                v-if="basecamera"
                :id="`stream-preview-${basecamera.name}`"
                class="mb-4 border border-white"
              />
            </template>
          </div>
        </div>
      </div>
      <div
        v-if="selectedItem === 'Discrete'"
        class="flex h-auto p-4"
      >
        <div class="grow">
          <v-radio
            label="Movement Mode"
            options="Straight, Spin"
            :selected="movementMode"
            @input="setMovementMode($event.detail.value)"
          />
          <div class="flex items-center gap-2 pt-4">
            <v-radio
              v-if="movementMode === 'Straight'"
              label="Movement Type"
              options="Continuous, Discrete"
              :selected="movementType"
              @input="setMovementType($event.detail.value)"
            />
            <v-radio
              v-if="movementMode === 'Straight'"
              label="Direction"
              options="Forwards, Backwards"
              :selected="direction"
              @input="setDirection($event.detail.value)"
            />
            <v-input
              v-if="movementMode === 'Straight'"
              type="number"
              :value="speed"
              label="Speed (mm/sec)"
              @input="speed = $event.detail.value"
            />
            <div
              v-if="movementMode === 'Straight'"
              :class="{ 'pointer-events-none opacity-50': movementType === 'Continuous' }"
            >
              <v-input
                type="number"
                :value="increment"
                :readonly="movementType === 'Continuous' ? 'true' : 'false'"
                label="Distance (mm)"
                @input="increment = $event.detail.value"
              />
            </div>
            <v-input
              v-if="movementMode === 'Spin'"
              type="number"
              :value="spinSpeed"
              label="Speed (deg/sec)"
              @input="spinSpeed = $event.detail.value"
            />
            <v-radio
              v-if="movementMode === 'Spin'"
              label="Movement Type"
              options="Clockwise, Counterclockwise"
              :selected="spinType"
              @input="setSpinType($event.detail.value)"
            />
            <div
              v-if="movementMode === 'Spin'"
              class="w-72 pl-6"
            >
              <v-slider
                :min="0"
                :max="360"
                :step="90"
                suffix="°"
                label="Angle"
                :value="angle"
                @input="angle = $event.detail.value"
              />
            </div>
          </div>
        </div>
        <div class="self-end">
          <v-button
            icon="play-circle-filled"
            variant="success"
            label="RUN"
            @click="baseRun()"
          />
        </div>
      </div>
    </div>
  </v-collapse>
</template>
