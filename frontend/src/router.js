import { createRouter, createWebHashHistory} from 'vue-router'
import Activities from './views/Activities.vue'
import Upload from './views/Upload.vue'
import Activity from './views/Activity.vue'

export default createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/activities' },
    { path: '/activities', component: Activities },
    { path: '/upload', component: Upload },
    { path: '/activities/:id', component: Activity, props: true },
  ],
})
