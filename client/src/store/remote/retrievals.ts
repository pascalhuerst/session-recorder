
import { ActionTree } from 'vuex';
//import { grpc } from 'grpc-web-client';
import {grpc} from "@improbable-eng/grpc-web";
import { OpenSessions, ReceiveOpenSessionsRequest, Session } from 'alsa2fifo_indexer/proto/ts/sessionsource_pb.d';
import { Sessionsource } from 'alsa2fifo_indexer/proto/ts/sessionsource_pb_service';
import { RootState } from '../types';


export const retrievals: ActionTree<RootState, RootState> = {
    connectStreams({ dispatch }) {
        dispatch('connectOpenSessions');
    },
    connectOpenSessions({ commit, dispatch }) {
        const req = new ReceiveOpenSessionsRequest();
        
        console.log("##### called: connectOpenSessions");
        
        grpc.invoke(Sessionsource.ReceiveOpenSessions, {
            request: req,
            host: 'localhost',
            onMessage: (message: OpenSessions) => {
                console.log("got open sessions: ", message.toObject());
            },
            onEnd: (code: grpc.Code, msg: string | undefined, trailers: grpc.Metadata) => {
                if (code == grpc.Code.OK) {
                  console.log("all ok")
                } else {
                  console.log("hit an error", code, msg, trailers);
                }
            }
        })
    }
}