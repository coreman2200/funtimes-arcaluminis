
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import { OrbitControls, Stats } from '@react-three/drei'
import * as THREE from 'three'
import create from 'zustand'

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

function VoxelCube(){
  const instRef = useRef<THREE.InstancedMesh>(null!)
  const top = useStore(s=>s.top)
  const colors = useStore(s=>s.colors)
  const isotropicZ = useStore(s=>s.isotropicZ)
  const normalizeCube = useStore(s=>s.normalizeCube)

  // Precompute positions in meters
  const { positions, scaleXYZ } = useMemo(()=>{
    const pts: [number,number,number][] = []
    const {x:X,y:Y,z:Z} = top.dim
    const stepXY = top.pitchMM
    const stepZ  = isotropicZ ? top.pitchMM : top.panelGapMM

    for (let z=0; z<Z; z++)
      for (let y=0; y<Y; y++)
        for (let x=0; x<X; x++){
          const px = (x - (X-1)/2) * stepXY / 1000
          const py = (y - (Y-1)/2) * stepXY / 1000
          const pz = (z - (Z-1)/2) * stepZ  / 1000
          pts.push([px,py,pz])
        }

    // Normalize to a cube: scale each axis so the outer extents match
    let scale: [number,number,number] = [1,1,1]
    if (normalizeCube && X>1 && Y>1 && Z>1){
      const spanX = (X-1)*stepXY
      const spanY = (Y-1)*stepXY
      const spanZ = (Z-1)*stepZ
      const maxSpan = Math.max(spanX, spanY, spanZ) || 1
      scale = [maxSpan/spanX || 1, maxSpan/spanY || 1, maxSpan/spanZ || 1]
    }
    return { positions: pts, scaleXYZ: scale }
  }, [top, isotropicZ, normalizeCube])

  useFrame(()=>{
    const m = new THREE.Matrix4()
    const c = new THREE.Color()
    for (let i=0; i<positions.length; i++){
      const [r,g,b] = [colors[i*3], colors[i*3+1], colors[i*3+2]]
      m.makeTranslation(...positions[i])
      instRef.current.setMatrixAt(i, m)
      c.setRGB(r/255, g/255, b/255)
      // @ts-ignore
      instRef.current.setColorAt(i, c)
    }
    instRef.current.instanceMatrix.needsUpdate = true
    // @ts-ignore
    instRef.current.instanceColor.needsUpdate = true
  })

  // Dot size: tie it to pitch so it looks good when normalized
  const dotRadiusM = Math.max(top.pitchMM, 1) / 1000 * 0.18

  return (
    <group scale={scaleXYZ as unknown as [number,number,number]}>
      <instancedMesh ref={instRef} args={[undefined, undefined, positions.length]}>
        <sphereGeometry args={[dotRadiusM, 6, 6]} />
        <meshBasicMaterial />
      </instancedMesh>
    </group>
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
    const sock = new WebSocket((location.origin.replace('http','ws')) + '/ws/control')
    sock.onopen = ()=> { sock.send(JSON.stringify({ runTest: name })); sock.close() }
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


function FirstRun(){
  const [open,setOpen] = React.useState(true)
  const [driver,setDriver] = React.useState<'sim'|'spi'>(()=> /Linux|arm/i.test(navigator.userAgent) ? 'spi' : 'sim')
  const [x,setX] = React.useState(5), [y,setY] = React.useState(26), [z,setZ] = React.useState(5)
  const [pitch,setPitch] = React.useState(17.6), [gap,setGap] = React.useState(25)
  const apply = ()=>{
    // send to backend
    const sock = new WebSocket((location.origin.replace('http','ws')) + '/ws/control')
    sock.onopen = ()=>{
      sock.send(JSON.stringify({ dim:{x,y,z}, pitchMM:pitch, panelGapMM:gap }))
      sock.close()
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

export default function App(){
  return (
    <div style={{width:'100vw', height:'100vh'}}>
      <Canvas camera={{position:[1.2,1.2,1.2], fov:55}}>
        <ambientLight />
        <VoxelCube/>
        <OrbitControls makeDefault />
        <Stats />
      </Canvas>
      <UI/>
      <WSClient/>
      <FirstRun/>
      <DiagClient/>{/* exactly one */}
    </div>
  )
}

