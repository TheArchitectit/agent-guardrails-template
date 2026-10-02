// WCAG contrast check for the palette pairs used in styles.css (light and dark).
const L=h=>{const c=[1,3,5].map(i=>parseInt(h.slice(i,i+2),16)/255).map(v=>v<=.03928?v/12.92:((v+.055)/1.055)**2.4);return .2126*c[0]+.7152*c[1]+.0722*c[2]}
const r=(a,b)=>{const[x,y]=[L(a),L(b)].sort((p,q)=>q-p);return(x+.05)/(y+.05)}
const sets={light:{paper:'#F8FAFC',ink:'#0F172A',advisory:'#B45309',checked:'#2563EB',blocking:'#15803D',fail:'#B91C1C',muted:'#475569'},
dark:{paper:'#0F172A',ink:'#F8FAFC',advisory:'#FBBF24',checked:'#93C5FD',blocking:'#86EFAC',fail:'#FCA5A5',muted:'#CBD5E1'}}
let bad=0
for(const[n,s]of Object.entries(sets))for(const k of Object.keys(s)){if(k==='paper')continue;const v=r(s[k],s.paper);const ok=v>=4.5;if(!ok)bad++;console.log(`${ok?'PASS':'FAIL'} ${n} ${k} on paper ${v.toFixed(2)}`)}
const btn=r('#FFFFFF','#2563EB');console.log(`${btn>=4.5?'PASS':'FAIL'} btn white on checked ${btn.toFixed(2)}`);if(btn<4.5)bad++
process.exit(bad?1:0)
