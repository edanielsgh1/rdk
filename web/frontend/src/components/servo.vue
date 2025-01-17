<script setup lang="ts">

import { grpc } from '@improbable-eng/grpc-web';
import type Servo from '../gen/proto/api/component/servo/v1/servo_pb.esm';
import servoApi from '../gen/proto/api/component/servo/v1/servo_pb.esm';
import { displayError } from '../lib/error';
import { rcLogConditionally } from '../lib/log';

interface Props {
  name: string
  status: Servo.Status.AsObject
  rawStatus: Servo.Status.AsObject
}

const props = defineProps<Props>();

const stop = () => {
  const req = new servoApi.StopRequest();
  req.setName(props.name);

  rcLogConditionally(req);
  window.servoService.stop(req, new grpc.Metadata(), displayError);
};

const move = (amount: number) => {
  const servo = props.rawStatus;
  const oldAngle = servo.positionDeg ?? 0;
  const angle = oldAngle + amount;

  const req = new servoApi.MoveRequest();
  req.setName(props.name);
  req.setAngleDeg(angle);
  window.servoService.move(req, new grpc.Metadata(), displayError);
};

</script>

<template>
  <div>
    <v-collapse
      :title="name"
      class="servo"
    >
      <v-breadcrumbs
        slot="title"
        crumbs="servo"
      />
      <v-button
        slot="header"
        label="STOP"
        icon="stop-circle"
        variant="danger"
        @click="stop"
      />
      <div class="border border-t-0 border-black p-4">
        <h3 class="mb-1 text-sm">
          Angle: {{ status.positionDeg }}
        </h3>
           
        <div class="flex gap-1.5">
          <v-button
            label="-10"
            @click="move(-10)"
          />
          <v-button
            label="-1"
            @click="move(-1)"
          />
          <v-button
            label="1"
            @click="move(1)"
          />
          <v-button
            label="10"
            @click="move(10)"
          />
        </div>
      </div>
    </v-collapse>
  </div>
</template>
