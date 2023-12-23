import { ActionTree } from 'vuex';
import { RemoteState } from './types';
import { RootState } from '../types';
import { retrievals } from './retrievals';

export const actions: ActionTree<RemoteState, RootState> = Object.assign({}, retrievals);