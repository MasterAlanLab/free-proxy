import { create } from "zustand";
import { api, waitJob } from "./api";

export type Node = {id:string; country:string; country_code:string; ip_address:string; owner:string; as_name:string; transport:string; ip_type:string; latency_ms:number; status:string};
type Page = {items:Node[]; total:number; limit:number; offset:number};
type State = {
  nodes: Node[]; total:number; page:number; pageSize:number; status:string; ipType:string; country:string; search:string;
  system:any; gateway:any; stats:any; settings:any; diagnostics:any; credentials:any; logs:any[]; selected:Set<string>; busy:boolean; error:string; logDate:string; logLevel:string; logModule:string;
  load:()=>Promise<void>; loadNodes:()=>Promise<void>; setFilter:(values:Partial<State>)=>void; run:(path:string, init?:RequestInit)=>Promise<void>; toggleSelected:(id:string)=>void;
};

export const useStore = create<State>((set, get) => ({
  nodes:[], total:0, page:1, pageSize:20, status:"", ipType:"", country:"", search:"", system:null, gateway:null, stats:null, logDate:"", logLevel:"", logModule:"",
  settings:null, diagnostics:null, credentials:null, logs:[], selected:new Set(), busy:false, error:"",
  load: async () => {
    try {
      const [system,gateway,stats,settings,diagnostics,credentials,logs] = await Promise.all([
        api("/system/status"), api("/gateway/status"), api("/pool/statistics"), api("/settings"), api("/system/diagnostics"), api("/auth/config"), api<{logs:any[]}>(`/logs?date=${encodeURIComponent(get().logDate)}&level=${encodeURIComponent(get().logLevel)}&module=${encodeURIComponent(get().logModule)}`),
      ]);
      set({system,gateway,stats,settings,diagnostics,credentials,logs:logs.logs,error:""});
      await get().loadNodes();
    } catch (error) { set({error:(error as Error).message}); }
  },
  loadNodes: async () => {
    const s=get(); const q=new URLSearchParams({limit:String(s.pageSize),offset:String((s.page-1)*s.pageSize)});
    if(s.status)q.set("status",s.status); if(s.ipType)q.set("ip_type",s.ipType); if(s.country)q.set("country",s.country); if(s.search)q.set("search",s.search);
    const page=await api<Page>(`/proxies?${q}`); set({nodes:page.items,total:page.total});
  },
  setFilter:(values)=>{set(values); queueMicrotask(()=>get().loadNodes().catch((e)=>set({error:e.message})));},
  run:async(path,init={method:"POST"})=>{set({busy:true,error:""});try{const job=await api<{id?:string}>(path,init);if(job?.id)await waitJob(job.id);await get().load();}catch(e){set({error:(e as Error).message});}finally{set({busy:false});}},
  toggleSelected:(id)=>set((s)=>{const selected=new Set(s.selected);selected.has(id)?selected.delete(id):selected.add(id);return{selected};}),
}));
