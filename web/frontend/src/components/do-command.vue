<script setup lang="ts">

import { computed, ref } from 'vue';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import type { Resource } from '../lib/resource';
import genericApi from '../gen/proto/api/component/generic/v1/generic_pb.esm';
import { toast } from '../lib/toast';

interface Props {
  resources: Resource[]
}

const props = defineProps<Props>();

const resources = computed(() => props.resources);

const selectedComponent = ref();
const input = ref();
const output = ref();
const executing = ref(false);

const doCommand = (name: string, command: string) => {
  const request = new genericApi.DoCommandRequest();

  request.setName(name);
  request.setCommand(Struct.fromJavaScript(JSON.parse(command)));

  executing.value = true;

  window.genericService.doCommand(request, (error, response) => {
    if (error) {
      toast.error(`Error executing command on ${name}: ${error}`);
      executing.value = false;
      return;
    }

    if (!response) {
      toast.error(`Invalid response when executing command on ${name}`);
      executing.value = false;
      return;
    }

    output.value = JSON.stringify(response?.getResult()?.toObject(), null, '\t');
    executing.value = false;
  });
};
</script>

<template>
  <v-collapse
    title="DoCommand()"
    class="doCommand"
  >
    <div class="h-full w-full border border-t-0 border-black p-4">
      <v-select
        label="Selected Component"
        placeholder="Null"
        :options="resources.map(({ name }) => name).join()"
        :value="selectedComponent"
        :disabled="executing ? 'true' : 'false'"
        @input="selectedComponent = $event.detail.value"
      />
      <div class="flex h-full w-full flex-row gap-2">
        <div class="h-full w-full">
          <p class="text-large">
            Input
          </p>
          <div class="h-[250px] w-full border border-black p-2">
            <v-code-editor
              language="json"
              value="{}"
              @input="input = $event.detail.value"
            />
          </div>
        </div>
        <div class="flex min-w-[90px] flex-col justify-center">
          <v-button
            variant="inverse-primary"
            :label="executing ? 'RUNNING...' : 'DO'"
            :disabled="!selectedComponent || !input || executing ? 'true' : 'false'"
            @click="doCommand(selectedComponent, input)"
          />
        </div>
        <div class="h-full w-full">
          <p class="text-large">
            Output
          </p>
          <div class="h-[250px] w-full border border-black p-2">
            <v-code-editor
              language="json"
              :value="output"
              readonly="true"
            />
          </div>
        </div>
      </div>
    </div>
  </v-collapse>
</template>
