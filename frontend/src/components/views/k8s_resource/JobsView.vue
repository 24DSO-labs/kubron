<template>
  <div>
    <ResourceListBase
      :items="filteredJobs"
      :search="search"
      :resourceTitle="'Jobs'"
      :emptyMessage="'No jobs found'"
      :resourceType="'jobs'"
      @refresh="refreshData"
      @search="(v) => search = v"
      @select="onSelect"
      @contextAction="onContextAction"
    />
    
  </div>
</template>

<script lang="ts" setup>
import { useK8s } from '../../../composables/useK8s'
import useResourceView from '../../../composables/useResourceView'
import ResourceListBase from '../ResourceListBase.vue'
import type { JobInfo } from '../../../types'

const { filteredJobs, refreshData, search } = useK8s()

const { onSelect, onContextAction, showDetail, selectedItem } = useResourceView('jobs')



</script>
