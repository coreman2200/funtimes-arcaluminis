
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import { OrbitControls, Stats } from '@react-three/drei'
import * as THREE from 'three'
import create from 'zustand'
import * as AppAPI from '../../wailsjs/go/main/App';

function TinyToolbar(){
  const [preview, setPreview] = React.useState(true);
  const [renderer, setRenderer] = React.useState<"ocean"|"calib"|"solid"|"grad">("ocean");
  const [preset, setPreset] = React.useState("CalmDawn");

  // universal knobs
  const [intensity, setIntensity] = React.useState(1.0);
  const [ev, setEV] = React.useState(3);
  const [gamma, setGamma] = React.useState(1.6);
  const [sat, setSat] = React.useState(1.0);
  const [speed, setSpeed] = React.useState(1.0); // your renderers read this as a phase scale

  // ocean short-list (the only per-renderer group we show for now)
  const [oTide, setOTide] = React.useState(0.22);
  const [oWave, setOWave] = React.useState(1.1);
  const [oFoam, setOFoam] = React.useState(0.18);
  const [oChop, setOChop] = React.useState(0.45);

  const applyGlobals = async ()=>{
    await AppAPI.SetParam("BaseIntensity", intensity);
    await AppAPI.SetParam("ExposureEV", ev);
    await AppAPI.SetParam("OutputGamma", 2.2);
    await AppAPI.SetParam("PreviewGamma", gamma);
    await AppAPI.SetParam("Saturation", sat);
    await AppAPI.SetParam("TimeScale", speed); // add: renderers multiply t by TimeScale
  };

  const applyOcean = async ()=>{
    await AppAPI.SetParam("TideAmp", oTide);
    await AppAPI.SetParam("WaveSpeed", oWave);
    await AppAPI.SetParam("Foaminess", oFoam);
    await AppAPI.SetParam("Choppiness", oChop);
  };

  const togglePreview = async ()=>{
    const next = !preview; setPreview(next);
    await AppAPI.UISetPreview(next);
  };

  const hydrateFromBackend = React.useCallback(async ()=>{
  try {
    const p = await AppAPI.GetParams();
    if (typeof p.BaseIntensity === 'number') setIntensity(p.BaseIntensity);
    if (typeof p.ExposureEV    === 'number') setEV(Math.round(p.ExposureEV));
    if (typeof p.PreviewGamma  === 'number') setGamma(p.PreviewGamma);
    if (typeof p.Saturation    === 'number') setSat(p.Saturation);
    if (typeof p.TimeScale     === 'number') setSpeed(p.TimeScale);
    if (typeof p.TideAmp       === 'number') setOTide(p.TideAmp);
    if (typeof p.WaveSpeed     === 'number') setOWave(p.WaveSpeed);
    if (typeof p.Foaminess     === 'number') setOFoam(p.Foaminess);
    if (typeof p.Choppiness    === 'number') setOChop(p.Choppiness);
  } catch(e) {
    console.warn("GetParams failed", e);
  }
}, []);


  const oceanPresets = ["CalmDawn","SunnyDay","Sunset","NightStorm"];
  const solidPresets = ["Red","Green","Blue","White","Black"];
  const gradPresets  = ["Rainbow","IndexSweep"];
  const calibPresets = ["PanelChanSweep"];

  const presets = renderer==="ocean" ? oceanPresets :
                  renderer==="grad"  ? gradPresets  :
                  renderer==="calib" ? calibPresets : solidPresets;

  // 1) Hydrate slider defaults from backend on mount
  React.useEffect(()=>{ hydrateFromBackend(); }, [hydrateFromBackend]);

  // 2) Debounced setter (so changes take effect without “Run”, but don’t spam)
  const debouncedSet = React.useRef<ReturnType<typeof setTimeout>|null>(null);
  const applyParam = (k:string, v:number)=>{
    if (debouncedSet.current) clearTimeout(debouncedSet.current);
    debouncedSet.current = setTimeout(()=>{ AppAPI.SetParam(k, v); }, 120);
  };

  // 3) Wire sliders to call applyParam on change
  //   Example: Intensity slider -> applyParam("BaseIntensity", intensity)
  //   You can keep your Range component, just pass a callback:
  const onIntensityChange = (n:number)=>{ setIntensity(n); applyParam("BaseIntensity", n); };
  const onEVChange        = (n:number)=>{ const m=Math.round(n); setEV(m); applyParam("ExposureEV", m); };
  const onGammaChange     = (n:number)=>{ setGamma(n); applyParam("PreviewGamma", n); };
  const onSatChange       = (n:number)=>{ setSat(n); applyParam("Saturation", n); };
  const onSpeedChange     = (n:number)=>{ setSpeed(n); applyParam("TimeScale", n); };

  // 4) Replace the Run button so it ONLY selects renderer/preset
  const run = async ()=>{
    await AppAPI.SeqCmd("stop");
    await AppAPI.UIRenderPreset(renderer, preset);
    await hydrateFromBackend(); // reflect any preset-set defaults
  };

  const resetAll = async ()=>{
    await AppAPI.UIRenderPreset(renderer, preset); // reset to current selection's defaults
    await hydrateFromBackend();               // refresh sliders to match
  };


  // 5) For ocean-only sliders:
  const onTide   = (n:number)=>{ setOTide(n); applyParam("TideAmp", n); };
  const onWave   = (n:number)=>{ setOWave(n); applyParam("WaveSpeed", n); };
  const onFoam   = (n:number)=>{ setOFoam(n); applyParam("Foaminess", n); };
  const onChop   = (n:number)=>{ setOChop(n); applyParam("Choppiness", n); };

  return (
    <div style={{
      position:'fixed', top:300, left:8, padding:'8px 10px',
      background:'rgba(20,20,24,.9)', color:'#ddd', borderRadius:12, zIndex:1000,
      display:'grid', gridAutoFlow:'row', gap:8, width:340
    }}>
      <div style={{display:'flex', gap:8}}>
        <select value={renderer} onChange={e=>{ setRenderer(e.target.value as any); }}>
          <option value="ocean">Ocean</option>
          <option value="calib">Calib</option>
          <option value="grad">Gradient</option>
          <option value="solid">Solid</option>
        </select>
        <select value={preset} onChange={e=>setPreset(e.target.value)}>
          {presets.map(p=><option key={p} value={p}>{p}</option>)}
        </select>
        <button onClick={run}>Run</button>
        <button onClick={resetAll}>Reset</button>
        <label style={{marginLeft:'auto'}}>
          <input type="checkbox" checked={preview} onChange={togglePreview}/> Preview
        </label>
      </div>

      {/* Universal small row */}
      <Row label="Intensity">
        <Range v={intensity} set={onIntensityChange} min={0} max={2} step={0.01}/>
      </Row>
      <Row label="Exposure EV">
        <Range v={ev} set={onEVChange} min={-4} max={6} step={1}/>
      </Row>
      <Row label="Preview Gamma">
        <Range v={gamma} set={onGammaChange} min={1.0} max={2.4} step={0.01}/>
      </Row>
      <Row label="Saturation">
        <Range v={sat} set={onSatChange} min={0} max={1} step={0.01}/>
      </Row>
      <Row label="Speed (TimeScale)">
        <Range v={speed} set={onSpeedChange} min={0} max={3} step={0.05}/>
      </Row>

      {/* Tiny per-render group (only shows when needed) */}
      {renderer==="ocean" && (
        <div style={{marginTop:2, paddingTop:6, borderTop:'1px solid rgba(255,255,255,.12)'}}>
          <div style={{opacity:.7, marginBottom:6}}>Ocean</div>
            <Row label="TideAmp"><Range v={oTide} set={onTide} min={0} max={0.5} step={0.01}/></Row>
            <Row label="WaveSpeed"><Range v={oWave} set={onWave} min={0.3} max={2.0} step={0.01}/></Row>
            <Row label="Foaminess"><Range v={oFoam} set={onFoam} min={0} max={1.0} step={0.01}/></Row>
            <Row label="Choppiness"><Range v={oChop} set={onChop} min={0} max={1.0} step={0.01}/></Row>
          <button onClick={applyOcean}>Apply Ocean</button>
        </div>
      )}
    </div>
  );
}

function Row({label, children}:{label:string, children:React.ReactNode}){
  return <div style={{display:'grid', gridTemplateColumns:'110px 1fr', alignItems:'center', gap:8}}>
    <div style={{opacity:.7}}>{label}</div>
    <div>{children}</div>
  </div>
}
function Range({v,set,min,max,step}:{v:number,set:(n:number)=>void,min:number,max:number,step:number}){
  return <input type="range" value={v} min={min} max={max} step={step}
    onChange={e=>set(parseFloat(e.target.value))} style={{width:'100%'}}/>;
}

type Order = 'XYZ'|'XZY'|'YXZ'|'YZX'|'ZXY'|'ZYX';
type Chan  = 'RGB'|'RBG'|'GRB'|'GBR'|'BRG'|'BGR';

function swizzle3(r:number,g:number,b:number, order:Chan){
  switch(order){
    case 'RGB': return [r,g,b];
    case 'RBG': return [r,b,g];
    case 'GRB': return [g,r,b];
    case 'GBR': return [g,b,r];
    case 'BRG': return [b,r,g];
    case 'BGR': return [b,g,r];
  }
}

function remapLinearIndex(i:number, X:number,Y:number,Z:number, src:Order /* fastest→slowest */){
  // Convert a source linear index (with 'src' fastest axis order)
  // to our target (XYZ fastest → x + X*(y + Y*z)).
  switch(src){
    case 'XYZ': return i;
    case 'XZY': { const x=i%X, t=Math.floor(i/X); const z=t%Z, y=Math.floor(t/Z); return x + X*(y + Y*z); }
    case 'YXZ': { const y=i%Y, t=Math.floor(i/Y); const x=t%X, z=Math.floor(t/X); return x + X*(y + Y*z); }
    case 'YZX': { const y=i%Y, t=Math.floor(i/Y); const z=t%Z, x=Math.floor(t/Z); return x + X*(y + Y*z); }
    case 'ZXY': { const z=i%Z, t=Math.floor(i/Z); const x=t%X, y=Math.floor(t/X); return x + X*(y + Y*z); }
    case 'ZYX': { const z=i%Z, t=Math.floor(i/Z); const y=t%Y, x=Math.floor(t/Y); return x + X*(y + Y*z); }
  }
}

const _colorBufferCache = new Map<number, Float32Array>();

// tiny helper
function b64ToU8(b64: string): Uint8Array {
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

const isWails = typeof window !== 'undefined' && !!(window as any).runtime;

type Topology = {
  dim: {x:number,y:number,z:number}
  panelGapMM:number
  pitchMM:number
  order?: {xFlipEveryRow:boolean,yFlipEveryPanel:boolean}
}

// add to your zustand store definition:
const useStore = create<{
  top: Topology
  colors: Uint8Array
  isotropicZ: boolean        // Z uses pitch (visual cube per voxel)
  normalizeCube: boolean     // scale whole cloud to a cube
  setTop: (t:Topology)=>void
  setColors: (c:Uint8Array)=>void
  setView: (v: Partial<{isotropicZ:boolean; normalizeCube:boolean}>)=>void
}>(set=> ({
  top: { dim:{x:5,y:26,z:5}, panelGapMM:25, pitchMM:17.6 },
  colors: new Uint8Array(5*26*5*3),
  isotropicZ: true,
  normalizeCube: true,
  setTop: (t)=> set({top:t, colors:new Uint8Array(t.dim.x*t.dim.y*t.dim.z*3)}),
  setColors: (c)=> set({colors:c}),
  setView: (v)=> set(v),
}))

function onRunTest(name: string) {
  if (isWails) {
    AppAPI.RunTest(name).catch(console.error);
  } else {
    // existing web path via WebSocket (your prior code)
    // runTestViaWebsocket(name)
  }
}

function onSetRenderer(name: string, preset = "") {
  if (isWails) AppAPI.SetRenderer(name, preset).catch(console.error);
  // else WS path if you keep it for web
}

function onSeq(cmd: "start"|"stop"|"pause"|"resume") {
  if (isWails) AppAPI.SeqCmd(cmd).catch(console.error);
}

function onExposure(ev: number) { if (isWails) AppAPI.SetParam("ExposureEV", ev); }
function onLimiter(enabled: boolean){ if (isWails) AppAPI.SetBool("LimiterOn", enabled); }

function IndexTestButton(){
  const top = useStore(s=>s.top);
  const setColors = useStore(s=>s.setColors);
  return (
    <button onClick={()=>{
      const {x:X,y:Y,z:Z} = top.dim;
      const N = X*Y*Z;
      const buf = new Uint8Array(N*3);
      for (let i=0;i<N;i++){
        // simple rainbow by index (HSV-ish)
        const t = i / N;
        const R = Math.floor(255 * (0.5 + 0.5*Math.sin(6.283*t + 0)));
        const G = Math.floor(255 * (0.5 + 0.5*Math.sin(6.283*t + 2.094)));
        const B = Math.floor(255 * (0.5 + 0.5*Math.sin(6.283*t + 4.188)));
        buf[i*3+0]=R; buf[i*3+1]=G; buf[i*3+2]=B;
      }
      setColors(buf);
    }}>Index Gradient (debug)</button>
  );
}


function VoxelCube(){
  const instRef = useRef<THREE.InstancedMesh>(null!)
  const top = useStore(s=>s.top)
  const colors = useStore(s=>s.colors)
  const isotropicZ = useStore(s=>s.isotropicZ)
  const normalizeCube = useStore(s=>s.normalizeCube)

  // --- shaders: read per-instance color & draw it ---
  const vtx = `
    attribute vec3 instanceColor;
    varying vec3 vColor;
    void main(){
      vColor = instanceColor;
      vec4 mvPosition = modelViewMatrix * instanceMatrix * vec4(position, 1.0);
      gl_Position = projectionMatrix * mvPosition;
    }
  `;
  const frg = `
    precision mediump float;
    varying vec3 vColor;
    void main(){
      gl_FragColor = vec4(vColor, 1.0);
    }
  `;

  // Precompute positions in meters
  const { positions, scaleXYZ } = useMemo(()=>{
    const pts: [number,number,number][] = []
    const {x:X,y:Y,z:Z} = top.dim
    const stepXY = top.pitchMM || 0
    const stepZ  = (isotropicZ ? top.pitchMM : top.panelGapMM) || 0

    const flipX = false;   // 👈 left↔right
    const flipZ = true;   // 👈 front↔back

    for (let z=0; z<Z; z++)
      for (let y=0; y<Y; y++)
        for (let x=0; x<X; x++){
          // visual coords (mirrored) but index (x,y,z) stays the same
          const vx = flipX ? (X - 1 - x) : x;
          const vy = y;
          const vz = flipZ ? (Z - 1 - z) : z;

          const px = (vx - (X-1)/2) * stepXY / 1000;
          const py = (vy - (Y-1)/2) * stepXY / 1000;
          const pz = (vz - (Z-1)/2) * stepZ  / 1000;
          pts.push([px,py,pz]);
        }

    let scale: [number,number,number] = [1,1,1]
    if (normalizeCube && X>1 && Y>1 && Z>1 && stepXY>0 && stepZ>0){
      const spanX = (X-1)*stepXY, spanY = (Y-1)*stepXY, spanZ = (Z-1)*stepZ
      const maxSpan = Math.max(spanX, spanY, spanZ) || 1
      scale = [maxSpan/(spanX||1), maxSpan/(spanY||1), maxSpan/(spanZ||1)]
    }
    return { positions: pts, scaleXYZ: scale }
  }, [top, isotropicZ, normalizeCube])

  // Place transforms & ensure instanceColor attribute exists (once per geometry change)
  useEffect(()=>{
    const mesh = instRef.current
    if (!mesh) return

    mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage)

    const m = new THREE.Matrix4()
    for (let i=0; i<positions.length; i++){
      m.makeTranslation(...positions[i])
      mesh.setMatrixAt(i, m)
    }
    mesh.instanceMatrix.needsUpdate = true

    // Create/attach instanceColor attribute on the GEOMETRY (shader reads this)
    const geo = mesh.geometry as THREE.InstancedBufferGeometry
    const need = positions.length * 3
    if (!geo.getAttribute('instanceColor') || (geo.getAttribute('instanceColor').count !== positions.length)) {
      geo.setAttribute(
        'instanceColor',
        new THREE.InstancedBufferAttribute(new Float32Array(need), 3)
      )
    }
  }, [positions])

  // Paint colors into geometry attribute whenever buffer changes
  useEffect(() => {
    const mesh = instRef.current;
    if (!mesh) return;

    const geo  = mesh.geometry as THREE.InstancedBufferGeometry;
    let attr   = geo.getAttribute('instanceColor') as THREE.InstancedBufferAttribute | undefined;
    const need = positions.length * 3;

    if (!attr || attr.count !== positions.length) {
      // (Re)create geometry attribute if missing or wrong size
      attr = new THREE.InstancedBufferAttribute(new Float32Array(need), 3);
      geo.setAttribute('instanceColor', attr);
    }

    // choose or reuse a Float32Array for the copy
    let f = _colorBufferCache.get(need);
    if (!f) { f = new Float32Array(need); _colorBufferCache.set(need, f); }

    if (!colors || colors.length < need) {
      // dim grey placeholder
      for (let i = 0; i < positions.length; i++) {
        f[i*3+0] = 0.06; f[i*3+1] = 0.06; f[i*3+2] = 0.06;
      }
    } else {
      // XYZ order, RGB channels — direct copy to float [0..1]
      for (let i = 0; i < positions.length; i++) {
        const r = colors[i*3+0] / 255;
        const g = colors[i*3+1] / 255;
        const b = colors[i*3+2] / 255;

        // boost + gamma (preview only)
        const R = r, G = g, B = b;

        f[i*3+0] = R; f[i*3+1] = G; f[i*3+2] = B;
      }
    }

    (attr.array as Float32Array).set(f);
    attr.needsUpdate = true;
  }, [colors, positions]);

  const dotRadiusM = Math.max(top.pitchMM, 1) / 1000 * 0.18

  return (
    <group scale={scaleXYZ as unknown as [number,number,number]}>
      <instancedMesh ref={instRef} args={[undefined, undefined, positions.length]}>
        <sphereGeometry args={[dotRadiusM, 6, 6]} />
        {/* Use a custom shader that reads geometry attribute 'instanceColor' */}
        <shaderMaterial
          vertexShader={vtx}
          fragmentShader={frg}
          uniforms={{}}
          // make sure no color management dims us
          toneMapped={false}
        />
      </instancedMesh>
    </group>
  )
}

function DebugButtons(){
  const top = useStore(s=>s.top);
  const setColors = useStore(s=>s.setColors);
  return (
    <div style={{position:'fixed', right:150, bottom:8, zIndex:999}}>
      <button onClick={()=>{
        // All white
        const n = top.dim.x * top.dim.y * top.dim.z;
        const buf = new Uint8Array(n*3);
        buf.fill(255);
        setColors(buf);
        console.log('[LightTest] set all white', n);
      }}>Light Test (white)</button>

      <button onClick={()=>{
        // Single voxel red at index 0
        const n = top.dim.x * top.dim.y * top.dim.z;
        const buf = new Uint8Array(n*3);
        buf[0] = 255; // R of first instance
        setColors(buf);
        console.log('[LightTest] set voxel 0 red');
      }} style={{marginLeft:8}}>Voxel 0 Red</button>
    </div>
  );
}

function LightTestButton(){
  const top = useStore(s=>s.top)
  const setColors = useStore(s=>s.setColors)
  return (
    <button onClick={()=>{
      const n = top.dim.x * top.dim.y * top.dim.z
      const buf = new Uint8Array(n*3)
      for (let i=0;i<n;i++){ buf[i*3+0]=255; buf[i*3+1]=255; buf[i*3+2]=255 }
      setColors(buf)
      console.log('[LightTest] set all white', n, 'voxels')
    }}>
      Light Test (all white)
    </button>
  )
}

function UI(){
  const top = useStore(s=>s.top)
  const setTop = useStore(s=>s.setTop)
  const isotropicZ = useStore(s=>s.isotropicZ)
  const normalizeCube = useStore(s=>s.normalizeCube)
  const setView = useStore(s=>s.setView)
  const [x,setX]=useState(top.dim.x), [y,setY]=useState(top.dim.y), [z,setZ]=useState(top.dim.z)
  const [gap,setGap]=useState(top.panelGapMM), [pitch,setPitch]=useState(top.pitchMM)
  const apply = ()=> setTop({dim:{x,y,z}, panelGapMM:gap, pitchMM:pitch, order: top.order})
  const runTest = (name:string)=> {
    if (isWails) {
      onRunTest(name);       // drive the Go engine via Wails bindings
      return;
    }
    try {
      const sock = new WebSocket((location.origin.replace('http','ws')) + '/ws/control')
      sock.onopen = ()=> { sock.send(JSON.stringify({ runTest: name })); sock.close() }
    } catch (e) { console.warn('runTest WS failed:', e) }
  }
  return (
    <div className="overlay" style={{position:'absolute', top:10, left:10, padding:10, background:'#0008', color:'white', borderRadius:8, display:'flex', gap:12}}>
      <div>
        <div style={{display:'grid', gridTemplateColumns:'auto auto', gap:6}}>
          <label>X (strips)</label><input type="number" value={x} onChange={e=>setX(parseInt(e.target.value))}/>
          <label>Y (leds/strip)</label><input type="number" value={y} onChange={e=>setY(parseInt(e.target.value))}/>
          <label>Z (panels)</label><input type="number" value={z} onChange={e=>setZ(parseInt(e.target.value))}/>
          <label>Panel gap (mm)</label><input type="number" value={gap} step={1} onChange={e=>setGap(parseFloat(e.target.value))}/>
          <label>Pitch (mm)</label><input type="number" value={pitch} step={0.1} onChange={e=>setPitch(parseFloat(e.target.value))}/>
        </div>
        <div style={{marginTop:8, display:'flex', gap:8}}>
          <button onClick={apply}>Apply in Viewer</button>
          <button onClick={()=>{
            if (isWails) return;
            const sock = new WebSocket((location.origin.replace('http','ws')) + '/ws/control')
            sock.onopen = ()=> { sock.send(JSON.stringify({ dim:{x,y,z}, panelGapMM:gap, pitchMM:pitch })); sock.close() }
          }}>Sync to Backend</button>
        </div>
      </div>
      <div>
        <div style={{fontWeight:700, marginBottom:4}}>Tests</div>
        <div style={{display:'grid', gridTemplateColumns:'auto auto', gap:6}}>
          <button onClick={()=>runTest("IndexSweep")}>Index Sweep</button>
          <button onClick={()=>runTest("GradRainbow")}>RGB</button>
          <button onClick={()=>runTest("SolidRed")}>Solid Red</button>
          <button onClick={()=>runTest("ProgramDemo")}>Program Demo</button>
          <button onClick={()=>runTest("PanelChanSweep")}>PanelChanSweep</button>
          <button onClick={()=>runTest("OceanDawn")}>OceanDawn</button>
          <DebugButtons/>
          <IndexTestButton/>
        </div>
      </div>
      <div style={{marginTop:8, display:'grid', gridTemplateColumns:'auto auto', gap:8}}>
        <label><input type="checkbox"
          checked={isotropicZ}
          onChange={e=>setView({isotropicZ: e.target.checked})}/> Z uses pitch (isotropic)</label>
        <label><input type="checkbox"
          checked={normalizeCube}
          onChange={e=>setView({normalizeCube: e.target.checked})}/> Normalize to cube</label>
      </div>
    </div>
  )
}


function WSClient(){
  if (isWails) return null; 
  const setTop = useStore(s=>s.setTop)
  const setColors = useStore(s=>s.setColors)
  useEffect(()=>{
    const sock = new WebSocket((location.origin.replace('http','ws')) + '/ws/frames')
    sock.onmessage = (ev)=>{
      try {
        const o = JSON.parse(ev.data)
        if (o.dim && o.panelGapMM){ setTop(o as any) }
        else if (o.rgb){ setColors(new Uint8Array(o.rgb)) }
      } catch {}
    }
    return ()=> sock.close()
  },[])
  return null
}

function PreviewClient(){
  const setTop = useStore(s=>s.setTop)
  const setColors = useStore(s=>s.setColors)
  const [freeze, setFreeze] = React.useState(false)
  const [dbg, setDbg] = React.useState<{hits:number,x?:number,y?:number,z?:number,len?:number,sum?:number, r?:number,g?:number,b?:number}>({hits:0})

  // 👉 tweak these 2 constants if output looks wrong
  const engineOrder: Order = 'XYZ'; // try 'ZXY' if slices look shuffled
  const chanOrder:   Chan  = 'RGB'; // try 'GRB' if primaries look rotated

  // somewhere top-level in App.tsx
  React.useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      // macOS: Cmd+Opt+I
      const isMac = navigator.platform.toLowerCase().includes('mac');
      const macCombo = isMac && e.metaKey && e.altKey && e.key.toLowerCase() === 'i';
      const winLinCombo = !isMac && e.ctrlKey && e.shiftKey && e.key.toLowerCase() === 'i';
      if (macCombo || winLinCombo) {
        const rt = (window as any)?.runtime;
        if (rt?.WindowOpenDevTools) {
          rt.WindowOpenDevTools();
        } else {
          console.warn('Wails runtime or WindowOpenDevTools not available');
        }
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);


  React.useEffect(()=>{
    const rt = (window as any)?.runtime
    if (!rt || typeof rt.EventsOn !== 'function') {
      console.warn('No Wails runtime')
      return
    }

    let first = true
    const off = rt.EventsOn('preview:frame', (p:any) => {
      try{
        if (freeze) return
        const X=p.x|0, Y=p.y|0, Z=p.z|0
        const bin = atob(p.rgb)
        const src = new Uint8Array(bin.length)
        for (let i=0;i<bin.length;i++) src[i] = bin.charCodeAt(i)
        const N = X*Y*Z
        if (src.length < N*3) {
          console.warn('preview too small', {have:src.length, need:N*3}); return
        }

        // --- diagnostics (first frame) ---
        if (first){
          first = false
          const head = Array.from(src.slice(0, 12))
          console.log('[preview first] dims', {X,Y,Z}, 'bytes', src.length, 'head', head)
        }

        // --- transform to XYZ/RGB ---
        const dst = new Uint8Array(N*3)
        let sum=0, Rh=0, Gh=0, Bh=0
        for (let i=0;i<N;i++){
          const sBase = i*3
          const [r0,g0,b0] = swizzle3(src[sBase], src[sBase+1], src[sBase+2], chanOrder)!
          const j = remapLinearIndex(i, X,Y,Z, engineOrder)  // j in our XYZ fastest indexing
          dst[j*3+0] = r0
          dst[j*3+1] = g0
          dst[j*3+2] = b0
          sum += r0 + g0 + b0
          Rh += r0; Gh += g0; Bh += b0
        }

        // sample 3 far-apart voxels
        const s0 = 0, s1 = Math.floor(N/2), s2 = N-1;
        const P = (i:number)=> [src[i*3+0], src[i*3+1], src[i*3+2]];
        const p0 = P(s0), p1 = P(s1), p2 = P(s2);
        if (first || (dbg.hits % 60) === 0) {
          console.log('[preview sample RGB]', {p0, p1, p2});
        }

        const snapshot = {dim:{x:X,y:Y,z:Z}, rgb: dst.slice(0)}
        ;(window as any).__cubePreview = snapshot
        setTop({ dim:{x:X,y:Y,z:Z}, panelGapMM:25, pitchMM:17.6, order: undefined })
        setColors(dst)
        setDbg(d=>({hits:d.hits+1, x:X,y:Y,z:Z, len:dst.length, sum, r:Rh, g:Gh, b:Bh}))
      }catch(e){ console.error('preview decode/transform', e) }
    })
    return ()=> { off && off() }
  }, [freeze])

  return (
    <div style={{position:'fixed',left:8,bottom:8,padding:'6px 8px',background:'rgba(0,0,0,.55)',color:'#0f0',font:'12px monospace',zIndex:999}}>
      hits:{dbg.hits} dim:{dbg.x}×{dbg.y}×{dbg.z} bytes:{dbg.len}
      <br/>Σ:{dbg.sum}  R:{dbg.r} G:{dbg.g} B:{dbg.b}
      <button style={{marginLeft:8}} onClick={()=>setFreeze(f=>!f)}>{freeze?'Resume':'Freeze'}</button>
    </div>
  )
}


function FirstRun(){
  
  const [open,setOpen] = React.useState(true)
  const [driver,setDriver] = React.useState<'sim'|'spi'>(()=> /Linux|arm/i.test(navigator.userAgent) ? 'spi' : 'sim')
  const [x,setX] = React.useState(5), [y,setY] = React.useState(26), [z,setZ] = React.useState(5)
  const [pitch,setPitch] = React.useState(17.6), [gap,setGap] = React.useState(25)
  const apply = ()=>{
    if (!isWails) {
      // send to backend
      const sock = new WebSocket((location.origin.replace('http','ws')) + '/ws/control')
      sock.onopen = ()=>{
        sock.send(JSON.stringify({ dim:{x,y,z}, pitchMM:pitch, panelGapMM:gap }))
        sock.close()
      }
    }
    setOpen(false)
  }
  if(!open) return null
  return (
    <div style={{position:'absolute', inset:0, background:'#000c', display:'grid', placeItems:'center'}}>
      <div style={{background:'#111', color:'#fff', padding:16, borderRadius:12, width:520}}>
        <h3 style={{marginTop:0}}>First‑Run Setup</h3>
        <p>Choose driver and confirm your cube dimensions. You can change these later in Settings.</p>
        <div style={{display:'grid', gridTemplateColumns:'auto auto', gap:8}}>
          <label>Driver</label>
          <select value={driver} onChange={e=>setDriver(e.target.value as any)}>
            <option value="spi">SPI</option>
            <option value="pwm">PWM (GPIO18)</option>
            <option value="sim">Simulator</option>
          </select>
          <label>X (strips)</label><input type="number" value={x} onChange={e=>setX(parseInt(e.target.value))}/>
          <label>Y (leds/strip)</label><input type="number" value={y} onChange={e=>setY(parseInt(e.target.value))}/>
          <label>Z (panels)</label><input type="number" value={z} onChange={e=>setZ(parseInt(e.target.value))}/>
          <label>Pitch (mm)</label><input type="number" value={pitch} step={0.1} onChange={e=>setPitch(parseFloat(e.target.value))}/>
          <label>Panel gap (mm)</label><input type="number" value={gap} step={1} onChange={e=>setGap(parseFloat(e.target.value))}/>
        </div>
        <div style={{marginTop:12, display:'flex', gap:8, justifyContent:'flex-end'}}>
          <button onClick={()=>setOpen(false)}>Skip</button>
          <button onClick={apply}>Save & Continue</button>
        </div>
      </div>
    </div>
  )
}


function DiagClient(){
  if (isWails) return null; 
  const [msgs,setMsgs] = React.useState<any[]>([])
  React.useEffect(()=>{
    const sock = new WebSocket((location.origin.replace('http','ws')) + '/ws/diag')
    sock.onmessage = (ev)=>{
      try{ const o = JSON.parse(ev.data); setMsgs(m=>[o, ...m].slice(0,50)) }catch{}
    }
    return ()=> sock.close()
  },[])
  return (
    <div style={{position:'absolute', right:10, top:10, maxWidth:420}}>
      {msgs.slice(0,5).map((m,i)=>(
        <div key={i} style={{background:'#111a', color:'#fff', padding:8, marginBottom:6, borderRadius:8, border:'1px solid #444'}}>
          <div style={{fontWeight:700}}>{m.summary || m.code}</div>
          {m.detail && <div style={{opacity:0.9, fontSize:12}}>{m.detail}</div>}
          {m.likely_causes && <div style={{fontSize:12, marginTop:4}}><b>Likely:</b> {m.likely_causes.join('; ')}</div>}
        </div>
      ))}
    </div>
  )
}

function RangeRow({
  label, value, setValue, min, max, step, format = (n)=>n.toFixed(2)
}: {
  label: string
  value: number
  setValue: (n:number)=>void
  min: number
  max: number
  step: number
  format?: (n:number)=>string
}) {
  return (
    <div>
      <div style={{display:'flex', justifyContent:'space-between'}}>
        <div style={{opacity:.8}}>{label}</div>
        <div>{format(value)}</div>
      </div>
      <input
        type="range"
        min={min} max={max} step={step}
        value={value}
        onChange={e => setValue(parseFloat(e.target.value))}
        style={{width:'100%'}}
      />
    </div>
  );
}


function CalibPanel(){
  const [open, setOpen] = React.useState(false);

  // ui state
  const [panelAxis, setPanelAxis] = React.useState(2);   // 0=X,1=Y,2=Z
  const [flipX, setFlipX] = React.useState(true);
  const [flipY, setFlipY] = React.useState(false);
  const [flipZ, setFlipZ] = React.useState(true);

  const [baseIntensity, setBaseIntensity] = React.useState(1.0);
  const [topWhitePow,  setTopWhitePow]  = React.useState(0.5);
  const [topWhiteMix,  setTopWhiteMix]  = React.useState(1.0);
  const [lrGamma,      setLRGamma]      = React.useState(1.6);
  const [rightFloor,   setRightFloor]   = React.useState(0.02);
  const [sat,          setSat]          = React.useState(1.0);
  const [gamma,        setGamma]        = React.useState(1.6);

  // post/limiter preview
  const [ev,           setEV]           = React.useState(3);
  const [whiteCap,     setWhiteCap]     = React.useState(3.0);
  const [knee,         setKnee]         = React.useState(1.0);
  const [gb,           setGB]           = React.useState(1.0);
  const [ledmA,        setLEDmA]        = React.useState(25);
  const [budget,       setBudget]       = React.useState(5000);

  const applyParams = async () => {
    if (!isWails) return;
    const P: Record<string, number> = {
      PanelAxis: panelAxis,
      FlipX: flipX ? 1 : 0,
      FlipY: flipY ? 1 : 0,
      FlipZ: flipZ ? 1 : 0,

      BaseIntensity: baseIntensity,
      TopWhitePow: topWhitePow,
      TopWhiteMix: topWhiteMix,
      LRGamma: lrGamma,
      RightFloor: rightFloor,
      Saturation: sat,
      Gamma: gamma,

      ExposureEV: ev,
      WhiteCap: whiteCap,
      LimiterKnee: knee,
      GlobalBrightness: gb,
      LEDChan_mA: ledmA,
      Budget_mA: budget,
      OutputGamma: 2.2,
    };
    for (const [k, v] of Object.entries(P)) {
      await AppAPI.SetParam(k, Number(v));
    }
  };

  const startCalib = async () => {
    if (!isWails) return;
    await AppAPI.SeqCmd("stop");                 // don’t let sequencer fight us
    await applyParams();
    await AppAPI.RunTest("PanelChanSweep");      // switch to calib renderer
  };

  const brightPreview = async () => {
    if (!isWails) return;
    // one-click “make it obvious” preview
    setEV(4); setWhiteCap(6); setKnee(1.0); setGB(1.0); setLEDmA(40); setBudget(20000);
    await AppAPI.SetParam("ExposureEV", 4);
    await AppAPI.SetParam("WhiteCap", 6);
    await AppAPI.SetParam("LimiterKnee", 1.0);
    await AppAPI.SetParam("GlobalBrightness", 1.0);
    await AppAPI.SetParam("LEDChan_mA", 40);
    await AppAPI.SetParam("Budget_mA", 20000);
  };

  return (
    <div style={{
      position:'fixed', top:8, right:8, width:320, padding:12,
      background:'rgba(20,20,24,.9)', color:'#ddd', font:'13px system-ui',
      borderRadius:12, boxShadow:'0 8px 24px rgba(0,0,0,.35)', zIndex:1000
    }}>
      <div style={{display:'flex', alignItems:'center', marginBottom:8}}>
        <strong style={{flex:1}}>Calib</strong>
        <button onClick={()=>setOpen(o=>!o)}>{open?'Hide':'Show'}</button>
      </div>
      {!open ? null : (
      <div style={{display:'grid', rowGap:8}}>
        <div>
          <div style={{marginBottom:6, opacity:.8}}>Panel axis</div>
          <div style={{display:'flex', gap:6}}>
            {[['X',0],['Y',1],['Z',2]].map(([lbl,val])=>(
              <label key={lbl as string} style={{display:'flex',gap:4,alignItems:'center'}}>
                <input type="radio" checked={panelAxis===val} onChange={()=>setPanelAxis(Number(val))}/>
                {lbl}
              </label>
            ))}
          </div>
          <div style={{marginTop:6, display:'flex', gap:10}}>
            <label><input type="checkbox" checked={flipX} onChange={e=>setFlipX(e.target.checked)}/> Flip X</label>
            <label><input type="checkbox" checked={flipY} onChange={e=>setFlipY(e.target.checked)}/> Flip Y</label>
            <label><input type="checkbox" checked={flipZ} onChange={e=>setFlipZ(e.target.checked)}/> Flip Z</label>
          </div>
        </div>

        <hr style={{opacity:.2}}/>

        <RangeRow label="BaseIntensity" value={baseIntensity} setValue={setBaseIntensity} min={0}   max={2}   step={0.01}/>
        <RangeRow label="TopWhiteMix"   value={topWhiteMix}   setValue={setTopWhiteMix}   min={0}   max={1}   step={0.01}/>
        <RangeRow label="TopWhitePow"   value={topWhitePow}   setValue={setTopWhitePow}   min={0.1} max={2.5} step={0.01}/>
        <RangeRow label="LRGamma"       value={lrGamma}       setValue={setLRGamma}       min={0.6} max={3.0} step={0.01}/>
        <RangeRow label="RightFloor"    value={rightFloor}    setValue={setRightFloor}    min={0}   max={0.2} step={0.005}/>
        <RangeRow label="Saturation"    value={sat}           setValue={setSat}           min={0}   max={1}   step={0.01}/>
        <RangeRow label="Gamma"         value={gamma}         setValue={setGamma}         min={1.0} max={2.4} step={0.01}/>

        <hr style={{opacity:.2}}/>

        <RangeRow label="ExposureEV" value={ev} setValue={(n)=>setEV(Math.round(n))} min={-4} max={6} step={1} format={(n)=>String(Math.round(n))}/>
        <RangeRow label="WhiteCap"   value={whiteCap} setValue={setWhiteCap} min={1}   max={10}    step={0.5}/>
        <RangeRow label="LimiterKnee"value={knee}      setValue={setKnee}     min={0.5} max={2.0}   step={0.1}/>
        <RangeRow label="GlobalBrightness" value={gb}  setValue={setGB}       min={0.25}max={1.5}   step={0.05}/>
        <RangeRow label="LEDChan_mA" value={ledmA}     setValue={(n)=>setLEDmA(Math.round(n))} min={5} max={60} step={1} format={(n)=>String(Math.round(n))}/>
        <RangeRow label="Budget_mA"  value={budget}    setValue={(n)=>setBudget(Math.round(n))} min={500} max={50000} step={500} format={(n)=>String(Math.round(n))}/>

        <div style={{display:'flex', gap:8, marginTop:6}}>
          <button onClick={brightPreview}>Bright Preview</button>
          <button onClick={applyParams}>Apply</button>
          <button onClick={startCalib} style={{marginLeft:'auto'}}>Run Calib</button>
        </div>
      </div>
      )}
    </div>
  );
}


export default function App(){
  return (
    <div style={{width:'100vw', height:'100vh'}}>
      <Canvas camera={{position:[1.2,1.2,1.2], fov:55}} style={{background:'#1a1a1a'}}>
        <ambientLight />
        <VoxelCube/>
        <OrbitControls makeDefault />
        <Stats className="stats-br" />
      </Canvas>
      <PreviewClient />
      <CalibPanel/>
      <TinyToolbar/>
      <UI/>
      <WSClient/>
      <FirstRun/>
      <DiagClient/>{/* exactly one */}
    </div>
  )
}
