import { Module } from 'vuex';
import { RootState } from '../types';
import { RemoteState } from './types';
import { actions } from './action';

export const remote: Module<RemoteState, RootState> = {
    actions,
 };