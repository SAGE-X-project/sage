"""Binarize UD-based candidates into an X0/X'/XP projection, retaining uncertainties.
Complement selection is heuristic. Multiple complements use explicit CompGroup,
not silently asserted grammatical X-bar structure. Punctuation is kept outside trees.
"""
import collections,gzip,json
from pathlib import Path
G=Path('analysis/2026-09-12/graphs'); counts=collections.Counter();samples=[]
for lang in ('en','ko'):
 with gzip.open(G/f'document-syntax-{lang}.jsonl.gz','rt') as inp,gzip.open(G/f'document-xbar-{lang}.jsonl.gz','wt',encoding='utf8') as out:
  for line in inp:
   row=json.loads(line)
   for sent in row['sentences']:
    base=sent['id'];nodes=[];edges=[];deps=collections.defaultdict(list);roots=[];uncertainties=[]
    tokens={w['id']:w for w in sent['tokens']}
    for w in sent['tokens']:
     if w['head']:deps[w['head']].append(w)
     elif w['upos']!='PUNCT':roots.append(f'{base}:t{w["id"]}:XP')
    def node(id,label,kind,**kw):nodes.append({'id':id,'label':label,'kind':kind,**kw});return id
    def edge(a,b,kind,**kw):edges.append({'source':a,'target':b,'kind':kind,**kw})
    cat={'NOUN':'N','PROPN':'N','PRON':'N','VERB':'V','AUX':'T','ADJ':'A','ADV':'Adv','ADP':'P','DET':'D','SCONJ':'C','CCONJ':'Conj','NUM':'Num'}
    for w in sent['tokens']:
     if w['upos']=='PUNCT':continue
     t=f'{base}:t{w["id"]}';c=cat.get(w['upos'],'X');xp=node(t+':XP',c+'P','projection');h=node(t+':head',w['text'],'head',category=c,token_id=w['id']);bar=node(t+':bar0',c+'′','bar');edge(bar,h,'head');current=bar
     ds=[d for d in deps[w['id']] if d['upos']!='PUNCT'];spec=[d for d in ds if d['deprel'].split(':')[0] in ('nsubj','csubj','det')];comp=[d for d in ds if d['deprel'].split(':')[0] in ('obj','iobj','ccomp','xcomp')];adj=[d for d in ds if d not in spec+comp]
     if comp:
      target=f'{base}:t{comp[0]["id"]}:XP'
      if len(comp)>1:
       uncertainties.append('multiple_complements_grouped_not_a_licensed_Xbar_derivation')
       for j,d in enumerate(comp[1:]):
        group=node(t+f':cg{j}','CompGroup','uncertain_group');edge(group,target,'member');edge(group,f'{base}:t{d["id"]}:XP','member');target=group
      edge(bar,target,'complement_candidate',confidence='heuristic')
     for j,d in enumerate(sorted(adj,key=lambda d:abs(d['id']-w['id']))):
      nb=node(t+f':bar{j+1}',c+'′','bar');edge(nb,current,'bar_continuation');edge(nb,f'{base}:t{d["id"]}:XP','adjunct_or_functional_candidate',ud_relation=d['deprel'],surface_side='left' if d['id']<w['id'] else 'right');current=nb
     if spec:
      # Single specifier is standard XP -> Spec X'. Multiple ones are explicit approximations.
      for j,d in enumerate(spec[:-1]):
       nb=node(t+f':specbar{j}',c+'′','uncertain_group');edge(nb,current,'bar_continuation');edge(nb,f'{base}:t{d["id"]}:XP','extra_specifier_candidate');current=nb;uncertainties.append('multiple_specifiers_approximated')
      edge(xp,f'{base}:t{spec[-1]["id"]}:XP','specifier_candidate',confidence='heuristic')
     edge(xp,current,'projects')
    sent['nodes']=nodes;sent['edges']=edges;sent['roots']=roots;sent['uncertainties']=sorted(set(uncertainties));counts['nodes']+=len(nodes);counts['edges']+=len(edges);counts['sentences']+=1;counts['uncertain_sentences']+=bool(uncertainties)
    degree=collections.Counter(e['source'] for e in edges);assert max(degree.values(),default=0)<=2
   row['method']='Stanza UD -> binary X0/Xbar/XP candidate projection; not gold linguistic constituency'
   out.write(json.dumps(row,ensure_ascii=False)+'\n')
   if (lang=='en' and len(samples)<10) or (lang=='ko' and len(samples)<20):samples.append(row)
(G/'xbar-sample.json').write_text(json.dumps(samples,ensure_ascii=False));(G/'xbar-summary.json').write_text(json.dumps(dict(counts),ensure_ascii=False,indent=2));print(dict(counts))
