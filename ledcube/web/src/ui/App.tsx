
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import { OrbitControls, Stats } from '@react-three/drei'
import * as THREE from 'three'
import create from 'zustand'
import * as AppAPI from '../../wailsjs/go/main/App';

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

    for (let z=0; z<Z; z++)
      for (let y=0; y<Y; y++)
        for (let x=0; x<X; x++){
          const px = (x - (X-1)/2) * stepXY / 1000
          const py = (y - (Y-1)/2) * stepXY / 1000
          const pz = (z - (Z-1)/2) * stepZ  / 1000
          pts.push([px,py,pz])
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
        f[i*3+0] = colors[i*3+0] / 255;
        f[i*3+1] = colors[i*3+1] / 255;
        f[i*3+2] = colors[i*3+2] / 255;
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
    <div style={{position:'fixed', right:50, bottom:8, zIndex:999}}>
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
        <div style={{display:'flex', gap:6}}>
          <button onClick={()=>runTest("index_sweep")}>Index Sweep</button>
          <button onClick={()=>runTest("rgb_channels")}>RGB</button>
          <button onClick={()=>runTest("plane_z")}>Plane Z</button>
          <LightTestButton/>
          <DebugButtons/>
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

function PreviewMosaic() {
  const canvasRef = React.useRef<HTMLCanvasElement>(null);

  React.useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return; // not mounted yet
    const ctx = canvas.getContext('2d');
    if (!ctx) return; // no 2d context available

    const off = document.createElement('canvas');
    const offctx = off.getContext('2d')!;
    const pad = 2;
    const cols = 4;

    let unsubscribe: (() => void) | undefined;
    const rt = (window as any)?.runtime;
    if (rt && typeof rt.EventsOn === 'function') {
      unsubscribe = rt.EventsOn('preview:frame', (p: any) => {
        try {
          const { x, y, z } = p;
          const bin = atob(p.rgb);
          const rgb = new Uint8Array(bin.length);
          for (let i = 0; i < bin.length; i++) rgb[i] = bin.charCodeAt(i);

          const cw = canvas.width;
          const ch = canvas.height;
          const tileW = Math.floor((cw - (cols - 1) * pad) / cols);
          const tileH = Math.floor(tileW * (y / x));

          ctx.clearRect(0, 0, cw, ch);
          off.width = x; off.height = y;

          let idx = 0;
          for (let zi = 0; zi < z; zi++) {
            const img = ctx.createImageData(x, y);
            for (let px = 0; px < x * y; px++) {
              img.data[4 * px + 0] = rgb[idx++];
              img.data[4 * px + 1] = rgb[idx++];
              img.data[4 * px + 2] = rgb[idx++];
              img.data[4 * px + 3] = 255;
            }
            offctx.putImageData(img, 0, 0);
            const col = zi % cols;
            const row = Math.floor(zi / cols);
            const dx = col * (tileW + pad);
            const dy = row * (tileH + pad);
            ctx.imageSmoothingEnabled = false;
            ctx.drawImage(off, dx, dy, tileW, tileH);
          }
        } catch (e) {
          console.error('preview:frame render error', e);
        }
      });
    }

    return () => { if (unsubscribe) unsubscribe(); };
  }, []);

  return <canvas ref={canvasRef} width={900} height={600} style={{ imageRendering: 'pixelated', border: '1px solid #333' }} />;
}


export default function App(){
  return (
    <div style={{width:'100vw', height:'100vh'}}>
      <Canvas camera={{position:[1.2,1.2,1.2], fov:55}}>
        <ambientLight />
        <VoxelCube/>
        <OrbitControls makeDefault />
        <Stats className="stats-br" />
      </Canvas>
      <PreviewClient />
      <UI/>
      <WSClient/>
      <FirstRun/>
      <DiagClient/>{/* exactly one */}
    </div>
  )
}

