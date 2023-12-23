import Vue from 'vue';
import Vuex, { StoreOptions } from 'vuex';
import { RootState } from './types';
import { remote } from './remote/index';

Vue.use(Vuex);

const store: StoreOptions<RootState> = {
  state: {
    version: '0.1.0',
  },
  modules: {
    remote,
  },
};

export default new Vuex.Store<RootState>(store);
