<script setup lang="ts">

import { grpc } from '@improbable-eng/grpc-web';
import sensorsApi from '../gen/proto/api/service/sensors/v1/sensors_pb.esm';
import commonApi from '../gen/proto/api/common/v1/common_pb.esm';
import { toast } from '../lib/toast';
import { resourceNameToString } from '../lib/resource';

interface SensorName {
  name: string
  namespace: string
  type: string
  subtype: string
}

interface Props {
  name: string
  sensorNames: SensorName[]
}

const props = defineProps<Props>();

interface Reading {
  _type: string
  lat: number
  lng: number
}

const sensorReadings = $ref<Record<string, Reading>>({});

const getReadings = (inputNames: SensorName[]) => {
  const req = new sensorsApi.GetReadingsRequest();
  const names = inputNames.map(({ name, namespace, type, subtype }) => {
    const resourceName = new commonApi.ResourceName();
    resourceName.setNamespace(namespace);
    resourceName.setType(type);
    resourceName.setSubtype(subtype);
    resourceName.setName(name);
    return resourceName;
  });
  req.setName(props.name);
  req.setSensorNamesList(names);

  window.sensorsService.getReadings(req, new grpc.Metadata(), (error, response) => {
    if (error) {
      return toast.error(error.message);
    }

    for (const item of response!.getReadingsList()) {
      const readings = item.getReadingsMap();
      const rr: Record<string, Reading> = {};

      for (const [k, v] of readings.entries()) {
        rr[k] = v.toJavaScript() as Reading;
      }
      
      sensorReadings[resourceNameToString(item.getName()!.toObject())] = rr;
    }
  });
};

</script>

<template>
  <v-collapse
    title="Sensors"
    class="sensors"
  >
    <div class="border border-t-0 border-black p-4">
      <table class="w-full table-auto border border-black">
        <tr>
          <th class="border border-black p-2">
            Name
          </th>
          <th class="border border-black p-2">
            Type
          </th>
          <th class="border border-black p-2">
            Readings
          </th>
          <th class="border border-black p-2 text-center">
            <v-button
              group
              label="Get All Readings"
              @click="getReadings"
            />
          </th>
        </tr>
        <tr
          v-for="sensorName in sensorNames"
          :key="sensorName.name"
        >
          <td class="border border-black p-2">
            {{ sensorName.name }}
          </td>
          <td class="border border-black p-2">
            {{ sensorName.subtype }}
          </td>
          <td class="border border-black p-2">
            <table style="font-size:.7em; text-align: left;">
              <tr
                v-for="(sensorValue, sensorField) in sensorReadings[resourceNameToString(sensorName)]"
                :key="sensorField"
              >
                <th>{{ sensorField }}</th>
                <td>
                  {{ sensorValue }}
                  <a
                    v-if="sensorValue._type == 'geopoint'"
                    :href="`https://www.google.com/maps/search/${sensorValue.lat},${sensorValue.lng}`"
                  >google maps</a>
                </td>
              </tr>
            </table>
          </td>
          <td class="border border-black p-2 text-center">
            <v-button
              group
              label="Get Readings"
              @click="getReadings([sensorName])"
            />
          </td>
        </tr>
      </table>
    </div>
  </v-collapse>
</template>
