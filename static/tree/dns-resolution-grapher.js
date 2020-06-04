const DNSResolutionGrapher = {};
(function(exports){
    // Definitions for node, edge, nodelist
    function Node(type,name, metadata={}){
        if(typeof(Node.id)==="undefined"){
            Node.id=0;
        }
        this.id = "n"+Node.id;
        this.type = type;
        this.uniqueName = `${type}~${name}`.toLowerCase()
        this.uniqueType = this.type;
        this.parents =[];
        this.biDirectionalNodes = [];
        this.children = [];
        if(this.type=="ip"){
            if(metadata.version!=null){
                this.uniqueType += metadata.version;
            }else if(metadata.accumulation){
                this.uniqueType+="_accumulation"
            }
            
        }else if(this.type=="nameserver"){
            if(metadata.archive){
                this.uniqueType += "_"+archive;
            }
        }
        this.name = name;
        this.metadata = metadata;
        this.addParent = function(node){
            let nodeFound = false;
            for(const oldNode of this.parents){
                if(node.uniqueName==oldNode.uniqueName){
                    nodeFound=true;
                    break;
                }
            }
            if(!nodeFound){
                this.parents.push(node);
                for(const oldNode of this.children){
                    if(node.uniqueName==oldNode.uniqueName){
                        nodeFound=true;
                        break;
                    }
                }
            }
        }
        this.addChild = function(node){
            let nodeFound = false;
            for(const oldNode of this.children){
                if(node.uniqueName==oldNode.uniqueName){
                    nodeFound=true;
                    break;
                }
            }
            if(!nodeFound){
                this.children.push(node);
            }
        }
        this.addBiDirectionalNode= function(node){
            let nodeFound = false;
            for(const oldNode of this.biDirectionalNodes){
                if(node.uniqueName==oldNode.uniqueName){
                    nodeFound=true;
                    break;
                }
            }
            if(!nodeFound){
                this.biDirectionalNodes.push(node);
            }
        }
        this.setTooltip = function(){
            if(this.metadata.accumulation
                || (this.metadata.hiddenTargets && this.metadata.hiddenTargets.length>0)
                || (this.metadata.hazard)
                || (this.metadata.warning && this.metadata.warning.length>0)
                || (this.metadata.preload && this.metadata.preload.length>0)){
                let accumulationNodes = this.metadata.substitutes || [];
                let hiddenTargets = this.metadata.hiddenTargets || [];
                let tooltipNodes = accumulationNodes.concat(hiddenTargets);
                this.metadata.tooltipNodes = tooltipNodes;
                this.metadata.ipCount=0;
                this.metadata.nameserverCount=0;
                let tooltipObj = {}
                for(const node of tooltipNodes){
                    if(node.type=="ip"){
                        this.metadata.ipCount++;
                    }else if(node.type=="nameserver"){
                        this.metadata.nameserverCount++;
                    }
                    if(tooltipObj[node.type.toUpperCase()]==null){
                        tooltipObj[node.type.toUpperCase()]=[];
                    }
                    tooltipObj[node.type.toUpperCase()].push(node.name);
                }
                this.metadata.tooltip = "<div>";
                let liColor = (this.metadata.preload) ? "#949494" : "#000000"
                Object.keys(tooltipObj).forEach((type)=>{
                    this.metadata.tooltip += `<b style="display:block;text-transform:uppercase;text-align:center">
                                        ${type}(s)</b>
                                        <hr style="margin:0; padding:0;">
                                    <ul style="padding-left:18px;margin-bottom:0;"><li style="color:${liColor}">${
                                        tooltipObj[type].join(`</li><li style="color:${liColor}">`)
                                    }</li></ul>`;
                });
                if(this.metadata.preload){
                    this.metadata.tooltip+=`<em style="display:block;">Database does not contain this domain</em>`;
                }
                if(this.metadata.hazard){
                    // Set hazard message to node-specific message
                    let hazardMessage=this.metadata.hazardMessage;
                    // Set hazard message to branch message
                    if(this.metadata.branch !=null && this.metadata.branch.hazardMessage!=null){
                        hazardMessage = this.metadata.branch.hazardMessage.join("<br>");
                    }
                    this.metadata.tooltip+=`<b style="display:block;color:#FF0000">${hazardMessage}</b>`;
                }
                if(this.metadata.warning){
                    // Set warning message to node-specific message
                    let warning=this.metadata.warning;
                    // Set warning message to branch message
                    if(this.metadata.branch !=null && this.metadata.branch.warning!=null){
                        warning = this.metadata.branch.warning.join("<br>");
                    }
                    this.metadata.tooltip+=`<b style="display:block;color:#CA9E2A">${warning}</b>`;
                }
                this.metadata.tooltip += "<div>";   
            }else{
                this.metadata.tooltip=null;
            }
        }
        this.setTooltip()
        ++Node.id;
    }
    function Edge(sourceNode, targetNode, metadata ={},checkBidirectional=true){
        if(typeof(Edge.id)==="undefined"){
            Edge.id=0;
        }
        this.id = "e"+Edge.id;
        let sourceDomain = (checkBidirectional) ? splitDomain(sourceNode.name) : undefined;
        let biDirectionalEdge = false;
        if(checkBidirectional && (sourceDomain.domain!=null || sourceDomain.tld!=null)){
            let targetDomain;
            if(targetNode.metadata.domain && targetNode.type!="ip"){
                targetDomain = splitDomain(targetNode.metadata.domain);
            }
            // If both target and source have the same domain or same tld and target isn't tld
            if((targetDomain!=null &&targetDomain.domain!=null && targetDomain.domain==sourceDomain.domain) || 
                (targetDomain!=null &&targetDomain.tld!=null && targetDomain.domain!=null 
                    && targetDomain.tld==sourceDomain.tld)){
                biDirectionalEdge=true;
            }
        }
        metadata.biDirectionalEdge = biDirectionalEdge;
        this.source = sourceNode.id;
        this.sourceNode = sourceNode;
        this.target = targetNode.id;
        this.targetNode = targetNode;
        this.metadata = metadata;
        ++Edge.id;
    }
    function NodeList(overrideMetadata={}){
        this.nodes=[];
        this.edges=[];
        this.levels=[];
        this.branches={};
        this.rootElement = null;
        // Default metadata
        this.overview = {};  
        // Callback function for onUpdate
        this.updateFunction = ()=>{};
        this.metadata={
            matchBranchColors:true,
            showUnmappedNodes:true,
            resolveZones:false,
            resolveArchive:false,
            accumulationNodes:["ip"],            
            hideNodes:["nameserver"],
            // Default nodelist formatting
            styleConfig:{
                "originX":0,
                "originY":40,
                "nodeMarginX":50,
                "nodeMarginY":100,
                "nodePadding":10,
                "nodeFontSize":16,
                "monospaceScaleFactor":0.6,
                "nodeFillColor":{
                    "domain":"#FFFFFF",
                    "ip6":"#e8c970",
                    "ip4":"#fddd82",
                    "ip":"#fddd82",
                    "ip_accumulation":"#fddd82",
                    "nameserver":"#d37b5f",
                    "nameserver_archive":"#d79580",
                    "zone":"#cbe870",
                    "default":"#FFCC00",
                },
                "edgeStrokeColor":{
                    "domain":"#000000",
                    "ip6":"#937a24",
                    "ip4":"#ac913b",
                    "ip":"#ac913b",
                    "ip_accumulation":"#ac913b",
                    "nameserver":"#d37b5f",
                    "nameserver_archive":"#d37b5f",
                    "zone":"#809e27",
                    "default":"#000000",
                },
                "nodeBorderRadius":{
                    "domain":0,
                    "ip6":0,
                    "ip4":0,
                    "ip":0,
                    "ip_accumulation":20,
                    "nameserver":0,
                    "nameserver_archive":0,
                    "zone":0,
                    "default":0,
                },
                "nodeBorderColor":"#000000",
                "nodeBorderWidth":"1.0",
                "nodeTextColor":"#000000",
                "edgeLineColor":"#000000",
                "edgeLineWidth":"2.0",
            }
        };
        NodeList.prototype.ipRegExp=new RegExp(/((^\s*((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\s*$)|(^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\s*$))/);
        Object.keys(overrideMetadata).forEach((key)=>{
            if(key in this.metadata){
                this.metadata[key] = overrideMetadata[key];
            }
        })
        this.updateOverview = function(property,nodeName){
            if(nodeName!=null){   
                this.overview[property] = this.overview[property] || [];
                if(!this.overview[property].includes(nodeName.toUpperCase())){
                    this.overview[property].push(nodeName.toUpperCase());
                }
                this.updateFunction(this.overview);
            }
        }
        // Create new nodelist with same properties as old nodelist
        this.newSublist = function(){
            const sublist = new NodeList(this.metadata);
            sublist.overview = this.overview;
            sublist.updateFunction = this.updateFunction;
            return sublist;
        }
        this.newListFromConfig = function(overrideMetadata, callback){
            const newNodeList = new NodeList(overrideMetadata);
            // Preserve colors
            newNodeList.branches = {};
            Object.keys(this.branches).forEach((branch)=>{
                newNodeList.branches[branch] = {
                    "color":this.branches[branch].color,
                    "nodes":[],
                }
            });
            if(callback!=null){
                if(typeof(callback)==='function'){
                    newNodeList.updateFunction = callback;
                }else{
                    throw Error("Invalid callback function");
                }
            }else{
                // Update callback defaults to empty function
                newNodeList.updateFunction = ()=>{};
            }
            // Remove all accumulation nodes and references to accumulation nodes
            const nodes = this.nodes.filter((node)=>!node.metadata.accumulation);
            nodes.forEach(node=>{
                // Unhide all nodes
                delete node.metadata.hidden;
                // Reset all accumulation lists
                node.metadata.accumulationNodes=[];
                node.metadata.hiddenTargets=[];
                node.metadata.tooltipNodes=[];
                // Reset hierarchical relationships
                node.children=[];
                node.parents=[];
                node.biDirectionalNodes=[];
                // Reset tooltip
                node.setTooltip();
            });
            const edges = this.edges.filter((edge)=>{
                let validEdge = true;
                // Remove all edges to accumulation nodes
                if(edge.sourceNode.metadata.accumulation || edge.targetNode.metadata.accumulation
                    || edge.metadata.sameTLD){
                    validEdge = false;
                }
                return validEdge;
            });
            edges.forEach(edge=>{
                // Unhide all edges
                delete edge.metadata.hidden;
            });

            newNodeList.add(...nodes);
            newNodeList.add(...edges);
            newNodeList.updateLevels();
            return newNodeList;
        }
        // Check if nodelist contains element
        this.contains = function(element){
            let containsElement = false;
            if(element instanceof Node){
                for(let i=0;i<this.nodes.length;++i){
                    if(this.nodes[i].uniqueName==element.uniqueName){
                        containsElement=this.nodes[i];
                        break;
                    }
                }
            }else if(element instanceof Edge){
                for(let i=0;i<this.edges.length;++i){
                    if(this.edges[i].source == element.source && this.edges[i].target == element.target){
                        containsElement = this.edges[i];
                        break;
                    }else if(this.edges[i].source == element.target && this.edges[i].target == element.source){
                        containsElement = this.edges[i];
                        break;
                    }
                }
            }else{
                throw Error("Invalid node");
            }
            return containsElement;
        }
        this.updateLevels = function(){
            // Auto adjust depth for nodes in relation to parent nodes
            // Each nodes depth should be equal to 1 + avg parent node depths
            let newLevels = [];
            let updatedNodes = [];
            let toUpdateNodes = new PriorityQueue((a,b)=>(a.positioning.priority||0)-(b.positioning.priority||0));
            let retryNodes = [];
            let maxToUpdateRepetitions = 10000;
            // # of levels up or down to look for empty spot
            let scanRadius=this.levels.length;
            toUpdateNodes.add(this.rootElement);
            while(!toUpdateNodes.empty() && maxToUpdateRepetitions>=0){
                let node = toUpdateNodes.remove();
                node.positioning= node.positioning||{};
                if(!updateDepth(node)){
                    let retryNames = retryNodes.map(node=>node.uniqueName);
                    let nodeIndex = retryNames.indexOf(node.uniqueName);
                    if(nodeIndex!=-1){
                        retryNodes.splice(nodeIndex,1);
                    }
                    retryNodes.push(node);
                }
                maxToUpdateRepetitions--;
            }
            // Sort by max viable parent depth ascending
            retryNodes.sort((a,b)=>{
                let maxParentDepthA =
                Math.max(...a.parents.filter((parent)=>!parent.metadata.hidden && parent.metadata.depth!=null)
                .map((parent)=>parent.metadata.depth),0);
                let maxParentDepthB =
                Math.max(...b.parents.filter((parent)=>!parent.metadata.hidden && parent.metadata.depth!=null)
                .map((parent)=>parent.metadata.depth),0);
                return a-b;
            });
            for(let i=0;i<retryNodes.length;i++){
                let missingParents = retryNodes[i].parents.filter((node)=> !node.metadata.hidden &
                    !updatedNodes.includes(node.uniqueName));
                missingParents.forEach((node)=>{
                    updateDepth(node,true);
                })
                updateDepth(retryNodes[i],true);
            }
            // Finalize formatting for newLevels
            // Truncate empty levels
            newLevels = newLevels.filter(level=>level!=null && level.length>0);
            // Capture initial row positions
            for(let i=0;i<newLevels.length;i++){
                // Inital sort is by number of children
                newLevels[i].sort((a,b)=>b.children.length-a.children.length);
                for(let j=0;j<newLevels[i].length;j++){
                    let node = newLevels[i][j];
                    //  Update depths according to truncated depth
                    node.metadata.depth=i;
                    // Unset node positioning
                    delete node.positioning;
                }
            }
            let overallPosition = {}; 
            for(let i=0;i<newLevels.length;i++){
                // Capture row positions within each row
                let innerRowPosition = {};
                let hasSameRowHierarchial = false;
                for(let j=0;j<newLevels[i].length;j++){
                    let node = newLevels[i][j];
                    // Get all of node's hierarchical nodes' names
                    let nodeHierarchy=node.children.concat(node.parents).map((node)=>node.uniqueName);
                    // If the hierarchical node is also in the current row, assign its position
                    let sameRowHierarchical = newLevels[i].filter((node)=>nodeHierarchy.includes(node.uniqueName));
                    if(sameRowHierarchical.length>0){
                        // Each node gets assigned a position within the row to minimize distance between
                        // hierarchical nodes. Node position is set to the assigned position, or to 3j+1
                        // to allow space to place adjacent elements on either side
                        if(innerRowPosition[node.uniqueName]==null){
                            innerRowPosition[node.uniqueName]= {"index":3*j+1,"position":"center"};
                        }
                        for(let k = 0;k<sameRowHierarchical.length;k++){
                            let sameRowNodeName = sameRowHierarchical[k].uniqueName;
                            let anchorIndex =  innerRowPosition[node.uniqueName].index;
                            let anchorPosition =  innerRowPosition[node.uniqueName].position;
                            // Alternate between left and right of current node for hierarchical nodes
                            if(innerRowPosition[sameRowNodeName]==null){
                                let sameRowIndex,sameRowPosition;
                                if(anchorPosition=="left"){
                                    sameRowIndex = anchorIndex-1;
                                    sameRowPosition = "left";
                                }else if(anchorPosition=="right"){
                                    sameRowIndex = anchorIndex+1;
                                    sameRowPosition = "right";
                                }else{  
                                    sameRowIndex = (k%2==0) ? anchorIndex+1 : anchorIndex-1;
                                    sameRowPosition = (k%2==0) ? "right" : "left";
                                }
                                innerRowPosition[sameRowNodeName] = {"index":sameRowIndex,"position":sameRowPosition}
                            }
                        }
                    }else{
                        let avgHierachicalIndex=0;
                        let viableHierarchical = nodeHierarchy.filter((nodeName)=>overallPosition[nodeName]!=null);
                        viableHierarchical.forEach((nodeName)=>{
                            avgHierachicalIndex+=overallPosition[nodeName];
                        });
                        if(viableHierarchical.length>0){
                            avgHierachicalIndex/=viableHierarchical.length;
                        }
                        // Make all non hierarchical nodes shift left of hierarchical nodes
                        avgHierachicalIndex += -10000;
                        innerRowPosition[node.uniqueName] = {"index":avgHierachicalIndex,"position":"none"};
                    }
                }
                // Sort nodes by intra-row position
                newLevels[i].sort((a,b)=>innerRowPosition[a.uniqueName].index-innerRowPosition[b.uniqueName].index);
                // Update overall position
                for(let j=0;j<newLevels[i].length;j++){
                    overallPosition[newLevels[i][j].uniqueName] = j;
                }
            }
            this.levels = newLevels;
            function updateDepth(node,forceAddNode=false){
                let canUpdateDepth = true;
                let maxAdjacentNodes = 2;
                if(!updatedNodes.includes(node.uniqueName)){
                    // Set default adjacent count for each node
                    node.positioning = node.positioning || {};
                    node.positioning.adjacent = node.positioning.adjacent || 0;
                    if(node.parents!=null && node.parents.length>0){
                        // Viable nodes are nodes that are not hidden, have
                        // valid depths, and have already been processed
                        let viableParents = node.parents.filter(
                        (parent)=>!parent.metadata.hidden && parent.metadata.depth!=null);
                        let visitedViableParents = viableParents.filter((parent)=>updatedNodes.includes(parent.uniqueName) ||
                            node.biDirectionalNodes.map(parent=>parent.uniqueName).includes(parent.uniqueName));
                        // Check if all viable nodes have been visited first or are bidirectional
                        if(visitedViableParents.length==viableParents.length && viableParents.length>0){
                            let parentDepths = getParentDepths(node);
                            // Calculate average viable node depth
                            let maxParentDepth = Math.max(...Object.keys(parentDepths).map(depth=>parseInt(depth)));   
                            node.metadata.depth=maxParentDepth;
                            updateAdjacents(node,maxAdjacentNodes);
                            // Update parent depths in case upadte adjacent changes depths
                            parentDepths = getParentDepths(node);
                            // Set inital adjacent value
                            let parentsAtDepth = parentDepths[node.metadata.depth];
                            // If the number of parents nodes adjacent at level is greater than max adjacent
                            // Or if the number of parents is greater than or equal to the max adjacent 
                            // due to parents sharing a parent limiting positioning options for new children
                            if(parentsAtDepth!=null && (parentsAtDepth.length>=maxAdjacentNodes
                            || Math.max(...parentsAtDepth.filter(parent=>parent.positioning && parent.positioning.adjacent!=null)
                                .map(parent=>parent.positioning.adjacent))>maxAdjacentNodes)){
                                let minNodesIndex = node.metadata.depth;
                                let minNodes = parentsAtDepth.length;
                                for(let i=1;i<=scanRadius;i++){
                                    // Allow no parents on same level if normalizing parents
                                    // If node depth > 0 and there are no adjacent nodes
                                    if(parentDepths[node.metadata.depth+i]==null || (parentDepths[node.metadata.depth+i].length<1
                                    && Math.max(...parentDepths[node.metadata.depth+i].filter(node=>node.positioning.adjacent!=null)
                                    .map(node=>node.positioning.adjacent),0)>maxAdjacentNodes-1)){
                                        minNodesIndex = node.metadata.depth+i;
                                        break;
                                    }else if(node.metadata.depth-i>0 && (parentDepths[node.metadata.depth-i]==null || 
                                    (parentDepths[node.metadata.depth-i].length<1
                                    && Math.max(...parentDepths[node.metadata.depth-i].filter(node=>node.positioning.adjacent!=null)
                                    .map(node=>node.positioning.adjacent),0)>maxAdjacentNodes-1))){
                                        minNodesIndex = node.metadata.depth-i;
                                        break;
                                    }
                                    // Else find min parents/children on same level
                                    if((parentDepths[node.metadata.depth-i]||[]).length<minNodes
                                        && node.metadata.depth-i>0){
                                        minNodes=(parentDepths[node.metadata.depth-i]||[]).length;
                                        minNodesIndex=node.metadata.depth-i;
                                    }
                                    if((parentDepths[node.metadata.depth+i]||[]).length<minNodes){
                                        minNodes=(parentDepths[node.metadata.depth+i]||[]).length;
                                        minNodesIndex=node.metadata.depth+i;
                                    }
                                }
                                node.metadata.depth = minNodesIndex;
                                updateAdjacents(node,maxAdjacentNodes);
                                parentDepths = getParentDepths(node);
                            }
                            // Update adjacent count on nodes at same level
                            (parentDepths[node.metadata.depth]||[]).forEach(parent=>{
                                parent.positioning = parent.positioning || {};
                                if(parent.positioning.adjacent!=null){
                                    parent.positioning.adjacent++;   
         
                                }else{
                                    parent.positioning.adjacent=0;
                                }
                            });
                        }else{
                            canUpdateDepth = false;
                        }
                    }else{
                        node.metadata.depth=0;
                        updateAdjacents(node);
                        node.positioning.priority = 0;
                    }
                    if((canUpdateDepth || forceAddNode)){
                        updatedNodes.push(node.uniqueName);
                        newLevels[node.metadata.depth]= newLevels[node.metadata.depth] || [];
                        newLevels[node.metadata.depth].push(node);
                    }
                    // Floor priority to remove +.1 for nonBidirectional Childrens
                    let parentPriority = Math.floor(node.positioning.priority);
                    // Sort by min viable parents
                    node.children.filter((child)=>!child.metadata.hidden && child.metadata.depth!=null).sort((a,b)=>{
                        let parentsLengthA = (a.parents!=null) ? 
                            a.parents.filter((node)=>!node.metadata.hidden && node.metadata.depth).length : 0;
                        let parentsLengthB = (b.parents!=null) ? 
                            b.parents.filter((node)=>!node.metadata.hidden && node.metadata.depth).length : 0;
                        return parentsLengthA-parentsLengthB;
                    }).forEach((child)=>{
                        let isBidirectional = node.biDirectionalNodes
                            .filter((node)=>!node.metadata.hidden && node.metadata.depth!=null)
                            .map((node)=>node.uniqueName).includes(child.uniqueName);
                        child.positioning = child.positioning || {};
                        let viableParents = child.parents.filter((node)=>!node.metadata.hidden && node.metadata.depth!=null);
                        let viableChildren = child.children.filter((node)=>!node.metadata.hidden && node.metadata.depth!=null);
                        let isSinkNode = viableParents.length==1 && viableChildren==0;
                        if(isBidirectional){
                            // Run biDirectionalNodes first as they have more variability
                            child.positioning.priority = parentPriority+1;
                        }else if(isSinkNode){
                            // Run sink nodes last as they have no variability
                            child.positioning.priority = parentPriority+1.2;
                        }else{
                            child.positioning.priority = parentPriority+1.1;
                        }
                        let retryNames = retryNodes.map(node=>node.uniqueName)
                        // Only add if node isn't already a retry node
                        if(!retryNames.includes(child.uniqueName)){
                            toUpdateNodes.add(child); 
                        }  
                    });
                    // Update the adjacent hierarchical nodes for the supplied node
                    function updateAdjacents(anchor,maxAdjacentNodes){
                        // Update adjacent count on biDirectionalNodes
                        let viableParents = anchor.parents.filter((anchor)=>!anchor.metadata.hidden 
                            && anchor.metadata.depth!=null);
                        let viableChildren = anchor.children.filter((anchor)=>!anchor.metadata.hidden 
                            && anchor.metadata.depth!=null);
                        let viableNodes =viableParents.concat(viableChildren
                            .filter((child)=>!viableParents.map((parent)=>parent.uniqueName).includes(child.uniqueName)))
                        let toUpdateNodes = [...viableNodes,anchor];
                        let retryNodes = [];
                        toUpdateNodes.forEach((node)=>{  
                            let nodeDepths = {};
                            let viableUpdateParents = node.parents.filter((node)=>!node.metadata.hidden 
                                && node.metadata.depth!=null);
                            let viableUpdateChildren = node.children.filter((node)=>!node.metadata.hidden 
                                && node.metadata.depth!=null);
                            let viableUpdateNodes =viableUpdateParents.concat(viableUpdateChildren
                                .filter((child)=>!viableUpdateParents.map((parent)=>parent.uniqueName).includes(child.uniqueName)))
                            viableUpdateNodes.forEach((node)=>{
                                nodeDepths[node.metadata.depth] = nodeDepths[node.metadata.depth] || []; 
                                nodeDepths[node.metadata.depth].push(node);
                            });
                            let nodesAtDepth = nodeDepths[node.metadata.depth]||[];
                            node.positioning = node.positioning || {};
                            node.positioning.adjacent = nodesAtDepth.length;
                            // If too many adjacent nodes, offset all not updated nodes
                            if(node.positioning.adjacent>maxAdjacentNodes){
                                // Move all not updated nodes off of adjacent spots that aren't the anchor node
                                viableUpdateParents.filter((parent)=>!updatedNodes.includes(parent.uniqueName)
                                    && parent.uniqueName!=anchor.uniqueName)
                                .forEach(parent=>{
                                    parent.metadata.depth=Math.min(...parent.children.map(child=>child.metadata.depth))-1;
                                    parent.metadata.depth=Math.max(parent.metadata.depth,0);
                                })
                                viableUpdateChildren.filter((child)=>!updatedNodes.includes(child.uniqueName)
                                    && child.uniqueName!=anchor.uniqueName)
                                .forEach(child=>{
                                    child.metadata.depth=Math.max(...child.parents.map(child=>child.metadata.depth))+1;
                                })
                            }
                        });
                    }
                    function getParentDepths(node){
                        let viableParents = node.parents.filter(
                        (parent)=>!parent.metadata.hidden && parent.metadata.depth!=null);
                        let parentDepths = {};
                        viableParents.forEach((parent)=>{
                            parentDepths[parent.metadata.depth] = parentDepths[parent.metadata.depth] || []; 
                            parentDepths[parent.metadata.depth].push(parent);
                        });
                        return parentDepths;
                    }
                }
                return canUpdateDepth;
            }
        }
        this.add=function(){
            [...arguments].forEach((element)=>{
                if(element instanceof Node){
                    let priorNode = this.contains(element)
                    if(!priorNode){
                        this.updateOverview("node",element.name);
                        // Insert either unique type or regular type into overview
                        if(element.type!=element.uniqueType){
                            this.updateOverview(element.uniqueType,element.name);
                        }else{
                            this.updateOverview(element.type,element.name);
                        }
                        if(this.nodes.length==0){
                            this.rootElement = element;
                        }
                        this.nodes.push(element);
                        // Default node depth if none specified
                        let nodeDepth = 0;
                        if(typeof(element.metadata.depth)!=="undefined"){
                            nodeDepth = element.metadata.depth;
                        }
                        // Only append node if not a hidden node
                        if(!this.metadata.hideNodes.includes(element.type) && !element.metadata.hidden){
                            if(typeof(this.levels[nodeDepth])==="undefined"){
                                this.levels[nodeDepth]=[];
                            }
                            this.levels[nodeDepth].push(element);
                            // Update visible nodes overview if not hidden and not accumulated
                            if(!this.metadata.accumulationNodes.includes(element.type) || 
                            (this.metadata.accumulationNodes.includes(element.type) && element.metadata.accumulation)){
                                this.updateOverview("visible",element.name);
                            }
                        }else{
                            element.metadata.hidden = true;
                        }
                        // If domain is set then add to a branch
                        if(element.metadata.domain !=null){
                            if(!this.branches[element.metadata.domain.toUpperCase()]){
                                this.branches[element.metadata.domain.toUpperCase()]={nodes:[]};
                            }
                            let currentBranch = this.branches[element.metadata.domain.toUpperCase()];
                            // Maintain hazard status across branch
                            // If current element is a hazard element
                            // then set all nodes in branch to hazard
                            if(element.metadata.hazard){
                                currentBranch.hazard=true;
                                // Append hazard message to branch
                                currentBranch.hazardMessage = currentBranch.hazardMessage || [];
                                if(element.metadata.hazardMessage!=null &&
                                    !currentBranch.hazardMessage.includes(element.metadata.hazardMessage)){
                                    currentBranch.hazardMessage.push(element.metadata.hazardMessage);
                                }
                                // Update stats for each hazard domains
                                this.updateOverview("hazard",element.metadata.domain);
                                currentBranch.nodes.forEach((node)=>{
                                    node.metadata.hazard=true;
                                    node.setTooltip();
                                })
                            }
                            if(element.metadata.warning){
                                // Append warning message to branch
                                currentBranch.warning = currentBranch.warning || [];
                                if(element.metadata.warning!=null &&
                                    !currentBranch.warning.includes(element.metadata.warning)){
                                    currentBranch.warning.push(element.metadata.warning);
                                }
                                // Update stats for each warning(misconfigured) domains
                                this.updateOverview("warning",element.metadata.domain);
                                currentBranch.nodes.forEach((node)=>{
                                    node.metadata.warning=currentBranch.warning;
                                    node.setTooltip();
                                })
                            }
                            // If current branch contains a warning node,
                            // then all incoming nodes are warningous
                            if(currentBranch.warning){
                                element.metadata.warning=currentBranch.warning;
                                // Update stats for each warning node
                                element.setTooltip();
                            }
                            // If current branch contains a hazard node,
                            // then all incoming nodes are hazardous
                            if(currentBranch.hazard){
                                element.metadata.hazard=true;
                                // Update stats for each hazard node
                                element.setTooltip();
                            }
                            // Maintain preload status across branch
                            if(currentBranch.preload){
                                element.metadata.preload=true;
                                element.setTooltip();
                            }else if(element.metadata.preload){
                                this.updateOverview("preload",element.metadata.domain);
                                currentBranch.preload=true;
                                currentBranch.nodes.forEach((node)=>{
                                    node.metadata.preload=true;
                                    node.setTooltip();
                                })
                            }
                            element.metadata.branch = currentBranch;
                            currentBranch.nodes.push(element);
                        }
                    }else{
                        const priorParentsNames = priorNode.parents.map(parent=>parent.uniqueName);
                        const priorChildrenNames = priorNode.children.map(child=>child.uniqueName);
                        const priorBiDirectionalNames = priorNode.biDirectionalNodes.map(node=>node.uniqueName);
                        // Remove duplicate reference from parents, children, and bidirectional
                        // Readd reference to original node
                        element.parents.forEach((parent)=>{
                            parent.children=parent.children.filter((node)=>node.uniqueName!=element.uniqueName);
                            parent.children.push(priorNode);
                            if(!priorParentsNames.includes(parent.uniqueName)){
                                priorNode.parents.push(parent);
                            }
                        });
                        element.children.forEach((child)=>{
                            child.parents=child.parents.filter((node)=>node.uniqueName!=element.uniqueName);
                            child.parents.push(priorNode);
                            if(!priorChildrenNames.includes(child.uniqueName)){
                                priorNode.children.push(child);
                            }
                        });
                        element.biDirectionalNodes.forEach((node)=>{
                            node.biDirectionalNodes=node.biDirectionalNodes.filter((node)=>node.uniqueName!=element.uniqueName);
                            node.biDirectionalNodes.push(priorNode);
                            if(!priorChildrenNames.includes(node.uniqueName)){
                                priorNode.biDirectionalNodes.push(node);
                            }
                        });
                        // If accumulation node combine substitutes
                        if(element.metadata.accumulation){
                            for(const newNode of element.metadata.substitutes){
                                let nodeFound = false;
                                for(const oldNode of priorNode.metadata.substitutes){
                                    if(oldNode.uniqueName ==newNode.uniqueName){
                                        nodeFound=true;
                                        break;
                                    }
                                }
                                if(!nodeFound){
                                    priorNode.metadata.substitutes.push(newNode);
                                }
                            }
                            // Update accumulation tooltip
                            if(priorNode.metadata.accumulation){
                                priorNode.setTooltip();
                            }
                        }
                    }
                }else if(element instanceof Edge){
                    let priorSource = this.contains(element.sourceNode);
                    let priorTarget = this.contains(element.targetNode);
                    // If nodes already exist for the given edge,
                    // replace the new node with the old ones
                    if(priorSource){
                        element.sourceNode = priorSource;
                        element.source = priorSource.id;
                    }
                    if(priorTarget){
                        element.targetNode = priorTarget;
                        element.target = priorTarget.id;
                    }
                    let sourceNode = element.sourceNode;
                    let targetNode = element.targetNode;
                    let sourceType = element.sourceNode.type;
                    let targetType = element.targetNode.type;
                    // Only push if edge doesn't point to itself
                    if(element.source != element.target){
                        let priorEdge = this.contains(element);
                        // Establish node hierarchy
                        targetNode.addParent(sourceNode);
                        sourceNode.addChild(targetNode);
                        if(element.metadata.biDirectionalEdge){
                            sourceNode.addParent(targetNode);
                            targetNode.addChild(sourceNode);
                            sourceNode.addBiDirectionalNode(targetNode);
                            targetNode.addBiDirectionalNode(sourceNode);
                        }
                        if(!priorEdge){
                            this.edges.push(element);
                            // Set hidden nodes
                            if(this.metadata.hideNodes.includes(sourceType) || 
                                this.metadata.hideNodes.includes(targetType)){
                                element.metadata.hidden=true;
                                // Initialize hidden targets
                                sourceNode.metadata.hiddenTargets = sourceNode.metadata.hiddenTargets || [];
                                sourceNode.metadata.hiddenSources = sourceNode.metadata.hiddenSources || [];
                                targetNode.metadata.hiddenSources = targetNode.metadata.hiddenSources || [];
                                targetNode.metadata.hiddenTargets = targetNode.metadata.hiddenTargets || [];
                                let targetFound = false;
                                for(const oldNode of sourceNode.metadata.hiddenTargets){
                                    if(oldNode.uniqueName == targetNode.uniqueName){
                                        targetFound=true;
                                        break;
                                    }
                                }
                                if(!this.metadata.accumulationNodes.includes(targetType) ||
                                    targetNode.metadata.accumulation){
                                    if(!targetFound){
                                        // Maintain hidden nodes at depth
                                        sourceNode.metadata.hiddenTargets.push(targetNode);
                                        targetNode.metadata.hiddenSources.push(sourceNode);
                                    }
                                    if(!this.metadata.hideNodes.includes(sourceType)){
                                        if(targetNode.children){
                                            // Rely on domain if available, since accumulation node names can't be split
                                            let sourceTLD = splitDomain(sourceNode.metadata.domain || sourceNode.name).tld;
                                            let targetTLD = splitDomain(targetNode.metadata.domain || targetNode.name).tld;
                                            // Only add edge if sourceNode and targetNode have same tld
                                            targetNode.children.forEach((child)=>{
                                                if(child.type!="ip" || (sourceTLD!=null && targetTLD!=null 
                                                    && sourceTLD==targetTLD)){
                                                    this.add(new Edge(sourceNode,
                                                        child,{"hidden":this.metadata.hideNodes.includes(child.type),
                                                        "sameTLD":true},
                                                     false));
                                                }
                                            })
                                        }
                                    }else if(this.metadata.hideNodes.includes(sourceNode.type)){
                                        sourceNode.metadata.hiddenSources.forEach((source)=>{
                                            if(this.metadata.hideNodes.includes(targetNode.type)){    
                                                if(!source.metadata.hiddenTargets.map(node=>node.uniqueName).includes(targetNode.uniqueName)){
                                                    source.metadata.hiddenTargets.push(targetNode);
                                                }
                                            }else{
                                                let sourceTLD = splitDomain(sourceNode.name).tld;
                                                let targetTLD = splitDomain(targetNode.name).tld;
                                                if(targetNode.type!="ip" || (sourceTLD!=null && targetTLD!=null 
                                                    && sourceTLD==targetTLD)){
                                                    this.add(new Edge(source,targetNode));
                                                }
                                            }
                                        })
                                    }
                                }
                                sourceNode.setTooltip();
                            }
                            // Set accumulation nodes
                            // If either source or target node is being accumulated and not already an accumulation node
                            if((this.metadata.accumulationNodes.includes(sourceType) && !sourceNode.metadata.accumulation) || 
                                (this.metadata.accumulationNodes.includes(targetType) && !targetNode.metadata.accumulation)){  
                                element.metadata.hidden=true;
                                // If target node is accumulutated
                                if(this.metadata.accumulationNodes.includes(targetType)){
                                    // If not initialized, create accumulationNodes array
                                    if(!targetNode.metadata.accumulationNodes || targetNode.metadata.accumulationNodes.length==0){
                                        targetNode.metadata.accumulationNodes = [];    
                                        if(targetType=="nameserver"){
                                            // Group by domain name
                                            let accumulationNode = new Node(targetType,targetNode.metadata.domain+"'S NAMESERVERS",
                                            {"depth":targetNode.metadata.depth,"accumulation":true,"substitutes":[targetNode],
                                                "domain":targetNode.metadata.domain.toUpperCase()});
                                            targetNode.metadata.accumulationNodes.push(accumulationNode);
                                            targetNode.metadata.hidden = true;
                                            this.add(accumulationNode);
                                            accumulationNode.setTooltip();
                                        }else if(targetType=="ip"){
                                            // Group by asn
                                            if(targetNode.metadata.asns && targetNode.metadata.asns.length>0){
                                                targetNode.metadata.asns.forEach((asn)=>{
                                                    let accumulationNode = new Node(targetType,"AS:"+asn,
                                                    {"depth":targetNode.metadata.depth,"accumulation":true,"substitutes":[targetNode]});
                                                    targetNode.metadata.accumulationNodes.push(accumulationNode);
                                                    targetNode.metadata.hidden = true;
                                                    this.add(accumulationNode);
                                                    accumulationNode.setTooltip();
                                                }); 
                                            }else if(!(this.metadata.hideNodes.includes(sourceType) || 
                                                this.metadata.hideNodes.includes(targetType) ||
                                                this.metadata.accumulationNodes.includes(sourceType))){
                                                // Unhide edge if no asns available and 
                                                // the edge doesn't contain a hidden type
                                                // and source node is not an asn
                                                element.metadata.hidden = false;
                                            }
                                        }

                                    }
                                    // If not already an accumulation node
                                    if(!targetNode.metadata.accumulation){
                                        // Create an edge for the each accumulation node
                                        targetNode.metadata.accumulationNodes.forEach((node)=>{
                                            this.add(new Edge(sourceNode,node,{},true));
                                        });
                                    }
                                    // Remove hidden nodes
                                    if(this.levels[targetNode.metadata.depth]){
                                        this.levels[targetNode.metadata.depth] = this.levels[targetNode.metadata.depth]
                                        .filter((node)=>!node.metadata.hidden);
                                    }
                                }
                                // If source node is accumulated but not an accumulation node
                                if(this.metadata.accumulationNodes.includes(sourceType) && !sourceNode.metadata.accumulation){
                                    // If the source node has not already been accumulated
                                    if(!sourceNode.metadata.accumulationNodes || sourceNode.metadata.accumulationNodes.length==0){
                                        sourceNode.metadata.accumulationNodes = [];
                                        if(sourceType=="nameserver"){
                                            // Group by domain name
                                            let accumulationNode = new Node(sourceType,sourceNode.metadata.domain+"'S NAMESERVERS",
                                            {"depth":sourceNode.metadata.depth,"accumulation":true,"substitutes":[sourceNode],
                                                "domain":sourceNode.metadata.domain.toUpperCase()});
                                            sourceNode.metadata.accumulationNodes.push(accumulationNode);
                                            sourceNode.metadata.hidden = true;
                                            this.add(accumulationNode);
                                            accumulationNode.setTooltip();
                                        }else if(sourceType=="ip"){
                                            // Group by asn
                                            sourceNode.metadata.asns.forEach((asn)=>{
                                                let accumulationNode = new Node(sourceType,asn,
                                                {"depth":sourceNode.metadata.depth,"accumulation":true,"substitutes":[sourceNode]});
                                                sourceNode.metadata.accumulationNodes.push(accumulationNode);
                                                sourceNode.metadata.hidden = true;
                                                this.add(accumulationNode);
                                                accumulationNode.setTooltip();
                                            })
                                        }
                                    }
                                    // If not already an accumulation node
                                    if(!sourceNode.metadata.accumulation){
                                        // Create an edge for each accumulation node
                                        sourceNode.metadata.accumulationNodes.forEach((node)=>{
                                            this.add(new Edge(node,targetNode,{},false));
                                        })
                                    }
                                }
                            }
                        }else if(priorEdge.source ==element.target && priorEdge.target ==element.source){
                            priorEdge.metadata.biDirectionalEdge =true;
                            sourceNode.addBiDirectionalNode(targetNode);
                            targetNode.addBiDirectionalNode(sourceNode);
                        }
                    }
                }else{
                    throw Error("Invalid element");
                }
            })
        }
        this.merge = function(parentNode,nodeList,checkBidirectional=true){
            this.add(...nodeList.nodes,...nodeList.edges)
            if(nodeList.rootElement!=null){
                this.add(new Edge(parentNode,nodeList.rootElement,{},checkBidirectional));
            }
        }
    }
    // Handle processing data from api or mapped links cache
    async function processDNSCoffeeResponse(response_json, depth, followNodeChain=true, mappedLinks, nodeData){
        let currentNode = null;
        let currentEdge = null;
        let promise = new Promise(async (resolve,reject)=>{
            let responsePromises = [];
            let data = response_json.data;
            let preload = response_json.preload;
            let hazard = data.hazard;
            let hazardMessage = data.hazardMessage;
            // Create Base Node
            let type = data.type;
            let name = data.name;
            currentNode = new Node(type,name,{"link":data.link,"depth":depth});
            // Maintain preload/hazard state from the pre-mapped links
            currentNode.metadata.preload = preload;
            currentNode.metadata.hazard = hazard;
            currentNode.metadata.hazardMessage = (hazard) ? hazardMessage : undefined;
            if(data.type=="domain"){
                // Push Domain Node
                currentNode.metadata.domain = name.toUpperCase();
                nodeData.add(currentNode);
                if(followNodeChain){
                    // Create and push zone node list
                    if(data.zone!=null){
                        let zoneLink = "/zones/"+data.zone.name;
                        // Add zone promise to promises to wait for before returning
                        if(nodeData.metadata.resolveZones){
                            let zonePromise = resolveNode(zoneLink,depth+1,mappedLinks, nodeData)
                            .then((zoneData)=>{
                                // Override type since dns coffee returns zones as type:domain
                                zoneData.rootElement.type="zone";
                                return zoneData;
                            }).catch((error)=>{
                                // If zone not available create placeholder node for zone
                                let zoneData = nodeData.newSublist();;
                                zoneData.add(new Node("zone",data.zone.name.toUpperCase(),
                                    {"link":zoneLink,"depth":depth+1}));
                                return zoneData;
                            }).then((zoneData)=>{
                                nodeData.merge(currentNode, zoneData);
                            });
                            responsePromises.push(zonePromise);
                        }else{
                            let zoneData = nodeData.newSublist();;
                            zoneData.add(new Node("zone",data.zone.name.toUpperCase(),
                                {"link":zoneLink,"depth":depth+1}));
                            nodeData.merge(currentNode, zoneData);
                        }
                    }
                    // Create nodes for all adjacent nameservers
                    let nameservers = (data.nameservers || []);
                    // Add archive property to all archive_nameservers
                    let archive_nameservers = (data.archive_nameservers || []).map((nameserver)=>{
                        nameserver.archive=true;
                        return nameserver;
                    });
                    // Combine archive and non achive nameservers for processing if resolveArchive is true
                    if(nodeData.metadata.resolveArchive){
                        nameservers = nameservers.concat(archive_nameservers);
                    }
                    // If nameserver will trigger subsequent unmapped chain, 
                    // resolve nameserver synchronously
                    currentNode.nameservers = currentNode.nameservers || [];
                    let mappedNameserverLinks = [];
                    for(const nameserver of nameservers){
                        try{  
                            let nameserverDomain=splitDomain(nameserver.name);
                            let currentDomain = splitDomain(currentNode.name);
                            if(!NodeList.prototype.ipRegExp.test(nameserver.name)){ 
                                let runSync = false;
                                let sameDomain = false;
                                let hasDomain =false;
                                let sameTLD = false;
                                let fetchLink = nameserver.link || "/nameservers/"+nameserver.name;
                                // If node type is domain and the domains are different
                                // Else if node type is is zone and zones are different
                                if(nameserverDomain.domain!=null){
                                    hasDomain=true;
                                    if(nameserverDomain.domain.toUpperCase() == currentNode.name.toUpperCase()){
                                        sameDomain = true;
                                        sameTLD = true;
                                    }else{
                                        fetchLink = "/domains/"+nameserverDomain.domain
                                    }
                                }
                                if(nameserverDomain.tld!=null && currentDomain.tld!=null &&
                                nameserverDomain.tld.toUpperCase() == currentDomain.tld.toUpperCase()){
                                    sameTLD = true;
                                }else if(!hasDomain){
                                    fetchLink = "/zones/"+nameserverDomain.tld;
                                }
                                // If not same domain and fetch link hasnt been mapped, run synchronously
                                if(!sameDomain && !mappedNameserverLinks.includes(fetchLink.toUpperCase())){
                                    runSync = true;
                                    mappedNameserverLinks.push(fetchLink.toUpperCase());
                                }
                                // If same domain process nameservers asynchronously
                                if(sameDomain){
                                    if(currentNode.metadata.preload){
                                        // If preload dont reresolve node
                                        const nextNodeData = nodeData.newSublist();
                                        nextNodeData.add(new Node("nameserver",nameserver.name,{"depth":depth+1,
                                            "zone":nameserverDomain.tld}));
                                        nextNodeData.rootElement.ips=[];
                                        processNameserverResponse(nodeData,nextNodeData,currentNode);
                                    }else{
                                        // Merge nameserver data with current nodeList asynchronously
                                        const nameserverPromise = resolveNode(fetchLink,depth+1,mappedLinks, nodeData)
                                        .then((nextNodeData)=>{
                                            processNameserverResponse(nodeData,nextNodeData,currentNode);
                                        })
                                        responsePromises.push(nameserverPromise);
                                    }
                                }else if(!runSync){
                                    // Merge nameserver data with current nodeList asynchronously
                                    // If doesnt have domain and not same tld process different tld
                                    // Else process domain
                                    if(!hasDomain && !sameTLD){
                                        const zonePromise = resolveNode(fetchLink,depth+1,mappedLinks, nodeData)
                                        .then((nextNodeData)=>{
                                            nodeData.merge(currentNode,nextNodeData);   
                                        })
                                        responsePromises.push(zonePromise);
                                    }else{
                                        // Add each authorative nameserver as child of domain
                                        const authNameserverNode = new Node("nameserver",nameserver.name,{"depth":depth+1,
                                            "zone":{"name":nameserverDomain.tld},"domain":nameserverDomain.domain.toUpperCase()})
                                        // Use seperate list to premerge nameserverData
                                        const nameserverData = nodeData.newSublist();
                                        nameserverData.add(authNameserverNode);
                                        // Only map domain if domain is not preloaded
                                        if(!currentNode.metadata.preload){    
                                            responsePromises.push(mapNameserverDomain(nameserverData,nameserverDomain.domain,
                                                nameserver).then(()=>{
                                                    // Merge auth nameserverData
                                                    nodeData.merge(currentNode,nameserverData);
                                                }));
                                        }else{
                                            nodeData.merge(currentNode,nameserverData);
                                        }
                                    } 
                                }else{
                                    // Merge nameserver data with current nodeList asynchronously
                                    // If sameTLD process different domain
                                    // Else process domain
                                    if(!hasDomain && !sameTLD){
                                        const nextNodeData = await resolveNode(fetchLink,depth+1,mappedLinks, nodeData);
                                        nodeData.merge(currentNode,nextNodeData);
                                    }else{
                                        // Add each authorative nameserver as child of domain
                                        const authNameserverNode = new Node("nameserver",nameserver.name,{"depth":depth+1,
                                            "zone":{"name":nameserverDomain.tld},"domain":nameserverDomain.domain.toUpperCase()})
                                        // Use seperate list to premerge nameserverData
                                        const nameserverData = nodeData.newSublist();
                                        nameserverData.add(authNameserverNode);
                                        // Only map domain if domain is not preloaded
                                        if(!currentNode.metadata.preload){
                                            await mapNameserverDomain(nameserverData,nameserverDomain.domain,nameserver);
                                        }
                                        nodeData.merge(currentNode,nameserverData);
                                    }
                                }
                                // If nameserver root domain is different from current domain, trace root
                                async function mapNameserverDomain(nameserverData,domainName, nameserver){
                                    const promise = new Promise(async (rootResolve,rootReject)=>{
                                        const nameserverLink = nameserver.link || "/nameservers/"+nameserver.name;
                                        const domainLink  ="/domains/"+domainName;
                                        // Check if zone is mapped
                                        const zoneData = mappedLinks["/API/ZONES"].data.zones;
                                        const zone = (nameserverDomain.tld) ? {"name":nameserverDomain.tld} : null
                                        const zoneMapped= zoneData.map(zone=>zone.zone.toUpperCase()).includes(zone.name.toUpperCase());
                                        // Check if root domain is either domain or nameserver
                                        let nameserverPromises = [];
                                        // Only run zone lookup if zone is mapped
                                        if(zoneMapped){
                                            nameserverPromises.push(resolveNode(domainLink,
                                            depth+1,mappedLinks, nameserverData).catch((error)=>{}));  
                                        }
                                        await Promise.allSettled(nameserverPromises).then((results)=>{
                                            return new Promise(async (resolve,reject)=>{
                                                let validUrl = false;
                                                for(const promise of results){
                                                    // Merge the first viable promise
                                                    if(promise.status=="fulfilled" && promise.value){
                                                        let rootDomainData = promise.value;
                                                        resolve(rootDomainData);
                                                        validUrl = true;
                                                        break;
                                                    }
                                                }
                                                // If neither the nameserver or domain resolution works,
                                                // then begin checking if invalid url
                                                if(!validUrl && nameserverData.metadata.showUnmappedNodes){
                                                    // Preload mapped links with domain data to circumvent fetch;
                                                    let hazard = !zone;
                                                    let hazardMessage = "Invalid TLD";
                                                    // Check if zone is part of database, if it is store
                                                    // response and set hazard settings
                                                    if(zone!=null && zoneMapped){
                                                        hazard = true;
                                                        hazardMessage="Potentially available for registration";
                                                    }
                                                    let newMappedLinks = JSON.parse(JSON.stringify(mappedLinks));
                                                    newMappedLinks[("/api"+domainLink).toUpperCase()] = 
                                                    {
                                                        "data":{
                                                            "type":"domain",
                                                            "link":domainLink,
                                                            "mapped":false,
                                                            "name":domainName.toUpperCase(),
                                                            "zone":zone,
                                                            "hazard":hazard,
                                                            "hazardMessage": (hazard) ? hazardMessage : null,
                                                            "nameservers":[{
                                                                "type":"nameserver",
                                                                "name":nameserver.name,
                                                                "link":nameserverLink,
                                                            }],
                                                        },
                                                        "preload":true
                                                    };
                                                    resolveNode(domainLink,depth+1,newMappedLinks,nameserverData)
                                                    .then((rootDomainData)=>{
                                                        resolve(rootDomainData);
                                                    }).catch(error=>{
                                                        reject(error);
                                                    });
                                                }else if(!nameserverData.metadata.showUnmappedNodes){
                                                    reject(results[0].reason || "Invalid response");
                                                }else{
                                                    reject("Invalid response")
                                                }
                                            });
                                        }).then((rootDomainData)=>{
                                            rootDomainData = rootDomainData || nameserverData.newSublist();
                                            let checkBidirectional=false;
                                            const authNameserverNode = nameserverData.rootElement;
                                            // If authNameserver is in domain zone file then relationship is biDirectional
                                            if(rootDomainData.nodes.map(node=>node.name).includes(authNameserverNode.name)){
                                                checkBidirectional=true;
                                            }
                                            nameserverData.merge(authNameserverNode, rootDomainData,checkBidirectional);
                                            rootResolve();
                                        }).catch((error)=>{rootReject(error)});
                                    });
                                    return promise;
                                }
                                // Define nameserver data resolution process at domain level
                                async function processNameserverResponse(nodeData,nextNodeData,currentNode){
                                    // Set nameserver archive status
                                    if(nameserver.archive){
                                        nextNodeData.rootElement.metadata.archive = true;
                                    }
                                    let nameserverDomain=splitDomain(nextNodeData.rootElement.name);
                                    // Set branch domain
                                    nextNodeData.rootElement.metadata.domain = nameserverDomain.domain.toUpperCase();
                                    for(const node of nextNodeData.rootElement.ips){
                                        node.metadata.domain = nameserverDomain.domain.toUpperCase();
                                    }
                                    currentNode.nameservers.push(nextNodeData.rootElement);
                                    nodeData.merge(currentNode,nextNodeData);
                                }
                            }else{
                                const authNameserverNode = new Node("nameserver",nameserver.name,{
                                    "depth":depth+1,
                                    "zone":{"name":currentDomain.tld},
                                    "domain":currentDomain.domain.toUpperCase(),
                                    "warning":`NS record '${nameserver.name}' with IPv4/6 address`
                                })
                                // Use seperate list to premerge nameserverData
                                const nameserverData = nodeData.newSublist();
                                nameserverData.add(authNameserverNode);
                                nodeData.merge(currentNode,nameserverData);
                            }
                        }catch(error){
                            reject(error);
                        }
                    }
                }
           }else if(data.type=="nameserver"){
                if(data.zone){
                    currentNode.metadata.zone = data.zone;
                }
                nodeData.add(currentNode);
                currentNode.ips=[];
                if(followNodeChain){
                    // Increment Depth for IP Nodes
                    ++depth;
                    // Consolidate ipv4 and ipv6 objects, if they exist  
                    let ipData = [...(data.ipv4 || []),...(data.ipv6 || [])];
                    let archive_ipData = [...(data.archive_ipv4 || []),...(data.archive_ipv6 || [])]
                    .map((ip)=>{
                        ip.archive = true;
                        return ip;
                    });
                    // Combine archive and non-archive data for processing if resolveArchive is true
                    if(nodeData.metadata.resolveArchive){
                        ipData = ipData.concat(archive_ipData);
                    }
                    // Add nodes for IPv4 and IPv6
                    ipData.forEach((ip)=>{
                        let ipNode = new Node(ip.type,ip.name,
                            {"depth":depth,"version":ip.version,"link":ip.link,"archive":ip.archive});
                        nodeData.nodes.push(ipNode);
                        currentNode.ips.push(ipNode);
                        let ipPromise = fetch("https://stat.ripe.net/data/network-info/data.json?resource="+ip.name)
                        .then((response)=>{
                            if(response.ok){
                                return response.json();
                            }else{
                                throw Error("Invalid response")
                            }
                        }).then((response_json)=>{
                            let data = response_json.data;
                            ipNode.metadata.asns = data.asns;  
                            if(ipNode.metadata.asns.length==0){
                                ipNode.metadata.hazard = true;
                                ipNode.metadata.hazardMessage = `${currentNode.name}'s IP ${ipNode.name} does not have an AS`;
                            } 
                            currentEdge = new Edge(currentNode,ipNode);
                            nodeData.edges.push(currentEdge);
                        }).catch((error)=>{throw Error("Invalid response")});
                        responsePromises.push(ipPromise);
                    })
                }
            }
            Promise.allSettled(responsePromises).then(()=>resolve(nodeData));
        });
        return promise;
    }
    // Fetches data from DNS Coffee and stores it in mapped links
    function loadDNSCoffeeResponse(currentLink, mappedLinks={}){
        let promise = new Promise(async (resolve,reject)=>{
            if(typeof(AxiosFetch)!=="undefined"){
                // Prepend baseUrl to links
                let baseUrl = "/api"
                let fetchLink = baseUrl+currentLink;
                let followNodeChain = !Object.keys(mappedLinks).includes(fetchLink.toUpperCase());
                let dnsCoffeeResponse = null
                if(followNodeChain){
                    dnsCoffeeResponse = await AxiosFetch.fetchData(fetchLink)
                    .then(response=>response.data).catch((error)=>{
                        return {"error":error};
                    });
                    mappedLinks[fetchLink.toUpperCase()]=dnsCoffeeResponse;
                    // Set dnsCoffeeResponse to null to terminate execution
                    if(dnsCoffeeResponse.error!=null){
                        dnsCoffeeResponse=null;
                    }
                }else{
                    // Only allow to continue query if response was valid
                    let cachedResponse = mappedLinks[fetchLink.toUpperCase()];
                    if(cachedResponse.error!=null){
                        dnsCoffeeResponse = null
                    }else{
                        dnsCoffeeResponse = cachedResponse;
                        // If data is preloaded then follow node chain
                        followNodeChain = !!mappedLinks[fetchLink.toUpperCase()].preload;
                    }
                }
                if(dnsCoffeeResponse!=null){
                    resolve({
                        "dnsCoffeeResponse": dnsCoffeeResponse,
                        "followNodeChain":followNodeChain,
                    });
                }else{
                    reject(Error(`Failed to reach: ${fetchLink}`));
                }      
            }else{
                reject(Error("Fetch API not available"));
            }
        })
        return promise;
    }
    // Recursively returns a nodelist of the zone
    // and ips associatied with a domain or nameserver
    async function resolveNode(currentLink,depth=0,mappedLinks={}, nodeList){
        let promise = new Promise(async function(resolve, reject){
            if(!currentLink){
                reject(Error("Invalid Node Link"));
            }else{
                let newNodeList = nodeList.newSublist();
                loadDNSCoffeeResponse(currentLink,mappedLinks).then(({dnsCoffeeResponse,followNodeChain})=>{
                    processDNSCoffeeResponse(dnsCoffeeResponse, depth, 
                    followNodeChain, mappedLinks, newNodeList).then((nodeData)=>{resolve(nodeData)});
                }).catch((error)=>{
                    reject(error);
                })
            }
        });
        return await promise;
    }
    function GraphMLDocument(){
        const doc = new Document();
        const xmlDeclaration = '<?xml version="1.0" encoding="UTF-8" standalone="no"?>';
        const gml = doc.implementation.createDocument("","",null);
        // Shortens creating custom GML Elements
        gml.createElementWithAttributes = function(type,attributes,parent=null){
            const element = gml.createElement(type);
            for(const attr in attributes){
                element.setAttribute(attr,attributes[attr]);
            }
            if(!!parent){
                parent.appendChild(element);
            }
            return element;
        }
        // Create root graphml element
        const root = gml.createElementWithAttributes("graphml",{
            "xmlns":"http://graphml.graphdrawing.org/xmlns",
            "xmlns:java":"http://www.yworks.com/xml/yfiles-common/1.0/java", 
            "xmlns:sys":"http://www.yworks.com/xml/yfiles-common/markup/primitives/2.0", 
            "xmlns:x":"http://www.yworks.com/xml/yfiles-common/markup/2.0", 
            "xmlns:xsi":"http://www.w3.org/2001/XMLSchema-instance", 
            "xmlns:y":"http://www.yworks.com/xml/graphml", 
            "xmlns:yed":"http://www.yworks.com/xml/yed/3", 
            "xsi:schemaLocation":`http://graphml.graphdrawing.org/xmlns
                http://www.yworks.com/xml/schema/graphml/1.1/ygraphml.xsd`,
        },gml);
        // Add formatting keys for yED graphML
        const keyList=[
            {"id":"d0","for":"port","yfiles.type":"portgraphics"},
            {"id":"d1","for":"port","yfiles.type":"portgeometry"},
            {"id":"d2","for":"port","yfiles.type":"portuserdata"},
            {"id":"d3","for":"node","attr.type":"string","attr.name":"url"},
            {"id":"d4","for":"node","attr.type":"string","attr.name":"description"},
            {"id":"d5","for":"node","yfiles.type":"nodegraphics"},
            {"id":"d6","for":"graphml","yfiles.type":"resources"},
            {"id":"d7","for":"edge","attr.type":"string","attr.name":"url"},
            {"id":"d8","for":"edge","attr.type":"string","attr.name":"description"},
            {"id":"d9","for":"edge","yfiles.type":"edgegraphics"},
        ];
        keyList.forEach((key)=>{
            gml.createElementWithAttributes("key",key,root);
        });
        const graph = gml.createElementWithAttributes("graph",{"id":"G","edgedefault":"directed"},root);
        // Append Resources
        const resourceDataElement = gml.createElementWithAttributes("data",{"key":"d6"},root);
        const resourcesElement = gml.createElementWithAttributes("y:Resources",{},resourceDataElement);
        function createShape({shape="rectangle",x,y,height,width,borderColor="#000000",borderWidth="1.0",
            fontSize="16",fillColor="#FFFFFF",textColor="#000000",id,text}){
            const nodeElement = gml.createElementWithAttributes("node",{"id":id},graph);
            const dataElement = gml.createElementWithAttributes("data",{"key":"d5"},nodeElement);
            const shapeNodeElement = gml.createElementWithAttributes("y:ShapeNode",{},dataElement);
            const geometryElement = gml.createElementWithAttributes("y:Geometry",{
                "height":height,
                "width":width,
                "x":x,
                "y":y,
            },shapeNodeElement);
            const fillElement = gml.createElementWithAttributes("y:Fill",{
                "color":fillColor,
                "transparent":"false",
            },shapeNodeElement);
            const borderStyleElement = gml.createElementWithAttributes("y:BorderStyle",{
                "color":borderColor,
                "raised":"false",
                "type":"line",
                "width":borderWidth,
            },shapeNodeElement);
            const nodeLabelElement = gml.createElementWithAttributes("y:NodeLabel",{
                "alignment":"center",
                "autoSizePolicy":"content",
                "fontFamily":"Monospaced",
                "fontSize":fontSize,
                "fontStyle":"plain",
                "hasBackgroundColor":"false",
                "hasLineColor":"false",
                "horizontalTextPosition":"center",
                "iconTextGap":"4",
                "modelName":"custom",
                "textColor":textColor,
                "verticalTextPosition":"bottom",
                "visible":"true",
            },shapeNodeElement);
            nodeLabelElement.innerHTML = text;
            const shapeElement = gml.createElementWithAttributes("y:Shape",{"type":shape},shapeNodeElement);
        }
        function createEdge({id,source,target,lineColor="#000000",lineType="line",
            lineWidth="2.0",biDirectionalEdge=false}){
            const edgeElement = gml.createElementWithAttributes("edge",{
                "id":id,
                "source":source,
                "target":target,
            },graph);
            const dataElement = gml.createElementWithAttributes("data",{"key":"d9"},edgeElement);
            const plEdgeElement = gml.createElementWithAttributes("y:PolyLineEdge",{},dataElement);
            const pathElement = gml.createElementWithAttributes("y:Path",{
                "sx":"0.0",
                "sy":"15.0",
                "tx":"0.0",
                "ty":"-15.0"
            },plEdgeElement);
            const lineStyleElement = gml.createElementWithAttributes("y:LineStyle",{
                "color":lineColor,
                "type":lineType,
                "width":lineWidth,
            },plEdgeElement);
            const arrowsElement = gml.createElementWithAttributes("y:Arrows",{
                "source":biDirectionalEdge?"standard":"none",
                "target":"standard"
            },plEdgeElement);
            const bendStyleElement = gml.createElementWithAttributes("y:BendStyle",{
                "smoothed":"false"
            },plEdgeElement);
        }
        function toString(){ 
            return xmlDeclaration + new XMLSerializer().serializeToString(gml);
        }
        return {gml,createShape,createEdge,toString}
    }
    // Converts nodelist to a yEd 3.19.1.1 style graphML String
    function graphMLStringFromNodeList(nodeList){
        nodeList = formatNodeList(nodeList);
        const gmlDoc = new GraphMLDocument();
        // Add node elements
        [].concat(...nodeList.levels.filter(level=>level!=null)).forEach((node)=>{
            // Create main shapes (rectangle/rounded rectangle) for each node
            gmlDoc.createShape({
                "shape":(node.uniqueType=="ip_accumulation") ? "roundrectangle" : "rectangle",
                "height":node.metadata.height,
                "width":node.metadata.width,
                "x":node.metadata.x,
                "y":node.metadata.y,
                "fillColor":node.metadata.fillColor,
                "borderColor":node.metadata.borderColor,
                "borderWidth":node.metadata.borderWidth,
                "fontSize":node.metadata.fontSize,
                "textColor":node.metadata.textColor,
                "text":node.name,
                "id":node.id
            });
            // Add tooltip for nameservers
            if(node.metadata.tooltip){
                // All preload nodes get question mark label
                if(node.metadata.nameserverCount>0 || node.metadata.preload){
                    const circleDiameter =  node.metadata.fontSize*1.5;  
                    gmlDoc.createShape({
                        "shape":"ellipse",
                        "height":circleDiameter,
                        "width":circleDiameter,
                        // GraphML Shapes are top left aligned
                        "x":parseInt(node.metadata.x)+parseInt(node.metadata.tooltipOffset.nameserver.node.x)
                            - parseInt(circleDiameter/2),
                        "y":parseInt(node.metadata.y)+parseInt(node.metadata.tooltipOffset.nameserver.node.y)
                            - parseInt(circleDiameter/2),
                        "fillColor":"#FFFFFF",
                        "borderColor":"#000000",
                        "borderWidth":"1.0",
                        "fontSize":node.metadata.fontSize,
                        "textColor":"#000000",
                        "text":(node.metadata.preload) ? "?" : node.metadata.nameserverCount,
                        "id":"n"+Node.id++,
                    });
                }
                // Add tooltip for IPs
                if(node.metadata.ipCount>0){
                    gmlDoc.createShape({
                        "shape":"ellipse",
                        "height":node.metadata.fontSize*1.5,
                        "width":node.metadata.fontSize*1.5,
                        // GraphML Shapes are top left aligned
                        "x":parseInt(node.metadata.x)+parseInt(node.metadata.tooltipOffset.ip.node.x)
                            - parseInt(node.metadata.fontSize*1.5/2),
                        "y":parseInt(node.metadata.y)+parseInt(node.metadata.tooltipOffset.ip.node.y)
                            - parseInt(node.metadata.fontSize*1.5/2),
                        "fillColor":"#FFFFFF",
                        "borderColor":"#000000",
                        "borderWidth":"1.0",
                        "fontSize":node.metadata.fontSize,
                        "textColor":"#000000",
                        "text":(node.metadata.preload) ? "?" : node.metadata.ipCount,
                        "id":"n"+Node.id++,
                    });
                }
                // Add Triangle for hazards
                if(node.metadata.hazard){
                    // Multiplier for equilateral triangle with visual similarity to circle in area
                    const triangleWidth = nodeList.metadata.styleConfig.nodeFontSize/2*1.5*1.333*1.5;
                    const triangleHeight = nodeList.metadata.styleConfig.nodeFontSize*1.5*1.333*1.5/1.732;
                    gmlDoc.createShape({
                        "shape":"triangle",
                        // Convert radius to width and height
                        "height":triangleWidth,
                        "width":triangleHeight,
                        // GraphML Shapes are top left aligned
                        "x":parseInt(node.metadata.x)-triangleWidth/2,
                        "y":parseInt(node.metadata.y)-triangleHeight/2,
                        "fillColor":"#FF0000",
                        "borderColor":"#000000",
                        "borderWidth":"1.0",
                        "fontSize":node.metadata.fontSize,
                        "textColor":"#FFFFFF",
                        "text":"!",
                        "id":"n"+Node.id++,
                    });
                }
            }
        });
        // Add edge elements
        nodeList.edges.filter((edge)=>!edge.metadata.hidden).forEach((edge)=>{
            gmlDoc.createEdge({
                "id":edge.id,
                "source":edge.source,
                "target":edge.target,
                "lineColor":edge.metadata.lineColor,
                "lineType":edge.metadata.lineType,
                "lineWidth":edge.metadata.lineWidth,
                "biDirectionalEdge":edge.metadata.biDirectionalEdge,
            });
        });
        return gmlDoc.toString();

    }
    // Adds positioning and sizing elements to a nodeList
    function formatNodeList(nodeList){
        // General config for formatting
        const styleConfig = nodeList.metadata.styleConfig;
        // Set branch colors
        if(nodeList.metadata.matchBranchColors){
            if(Color!=null){
                Object.keys(nodeList.branches).forEach((branchName)=>{
                    const branch = nodeList.branches[branchName];
                    const branchColor = Color.randomFromString(branchName);
                    branch.color = branchColor.hexString;
                    branch.textColor = Color.contrastTextColor(branchColor.hexString,"hex");
                })
            }else{
                throw Error("Dependecy Color not loaded");
            }
        }
        // Initialize nodeList dimensions
        nodeList.metadata.height = styleConfig.originX+nodeList.levels.length*(styleConfig.nodeFontSize
            +styleConfig.nodePadding*2) + (nodeList.levels.length+1)*styleConfig.nodeMarginY;
        nodeList.metadata.width = styleConfig.originY;
        // Assign node specific metadata for formatting and positioning
        for(let i=0;i<nodeList.levels.length;++i){
            const nodes = nodeList.levels[i];
            if(nodes!=null){
                const nodeCharacters = nodes.map((node)=>node.name.length);
                const layerCharCount = nodeCharacters.reduce((acc,curr)=>acc+curr);
                const characterWidth = styleConfig.nodeFontSize*styleConfig.monospaceScaleFactor;
                // Layer Width = width of total letters + total node padding + inner node margin
                const layerWidth = layerCharCount*characterWidth+(nodes.length-1)*styleConfig.nodeMarginX + 
                nodes.length*(styleConfig.nodePadding*2);
                // Update nodelist size metadata
                nodeList.metadata.width = Math.max(layerWidth+styleConfig.nodeMarginX*2, 
                    (nodeList.metadata.width))
                // initalize current node offset
                let currentNodeOffset = 0;
                for(let j=0;j<nodes.length;++j){
                    const node = nodes[j];
                    const nodeWidth = (node.name.length*characterWidth+2*styleConfig.nodePadding);
                    const nodeHeight= (styleConfig.nodeFontSize+2*styleConfig.nodePadding);
                    node.metadata.width = nodeWidth.toFixed(1);
                    node.metadata.height = nodeHeight.toFixed(1);
                    // Positioning coordinates are at top left of node
                    node.metadata.x = (styleConfig.originX - layerWidth/2 + 
                        currentNodeOffset).toFixed(1);
                    node.metadata.y = (styleConfig.originY + 
                        i*(nodeHeight+styleConfig.nodeMarginY)).toFixed(1);
                    // Update node offset
                    currentNodeOffset+=nodeWidth+styleConfig.nodeMarginX;
                    // Positioning coordinates for segmented line
                    node.metadata.pointX = node.metadata.x;
                    node.metadata.pointY = (parseFloat(node.metadata.y) + (nodeHeight+
                        styleConfig.nodeMarginY)/2).toFixed(1);
                    // Initialize tooltip offset
                    node.metadata.tooltipOffset = {
                        "ip":{},
                        "nameserver":{}
                    };
                    // IP nodes offset to bottom right of node
                    node.metadata.tooltipOffset.ip.node={
                        "x":parseInt(node.metadata.width),
                        "y":parseInt(node.metadata.height),
                    };
                    // Position text 2 pixels higher than center to adjust for baseline
                    node.metadata.tooltipOffset.ip.text={
                        "x":parseInt(node.metadata.width),
                        "y":parseInt(node.metadata.height)+2,
                    };
                    // Nameserver tooltip offset to top right of node
                    node.metadata.tooltipOffset.nameserver.node={
                        "x":parseInt(node.metadata.width),
                        "y":0,
                    };
                    // Position text 2 pixels higher than center to adjust for baseline
                    node.metadata.tooltipOffset.nameserver.text={
                        "x":parseInt(node.metadata.width),
                        "y":2,
                    };
                    // Node styles
                    // Use type specific style or default
                    if(!nodeList.metadata.matchBranchColors || 
                        !(node.metadata.domain!=null && nodeList.branches[node.metadata.domain.toUpperCase()] != null)
                    ){
                        node.metadata.fillColor =  styleConfig.nodeFillColor[node.uniqueType] 
                            || styleConfig.nodeFillColor['default'];
                        node.metadata.textColor = styleConfig.nodeTextColor;
                    }else{
                        node.metadata.fillColor =  nodeList.branches[node.metadata.domain.toUpperCase()].color || 
                        styleConfig.nodeFillColor[node.uniqueType] || styleConfig.nodeFillColor['default'];
                        node.metadata.textColor = nodeList.branches[node.metadata.domain.toUpperCase()].textColor 
                            || styleConfig.nodeTextColor;
                    }
                    node.metadata.borderColor = styleConfig.nodeBorderColor;
                    node.metadata.borderWidth = styleConfig.nodeBorderWidth;
                    node.metadata.fontSize = styleConfig.nodeFontSize;
                    node.metadata.borderRadius = styleConfig.nodeBorderRadius[node.uniqueType] 
                    || styleConfig.nodeBorderRadius['default'];

                }
            }
        }
        // Assign edge specific metadata for formatting and positioning
        for(const edge of nodeList.edges){
            if(!edge.metadata.hidden){
                // Randomize source and target x coordinates
                if(edge.sourceNode.metadata.depth<edge.targetNode.metadata.depth){
                    // Edges are bottom aligned from the source and top aligned from the target
                    edge.metadata.sx = parseInt(edge.sourceNode.metadata.x)+parseInt(edge.sourceNode.metadata.width)/2;
                    edge.metadata.sy = parseInt(edge.sourceNode.metadata.y)+parseInt(edge.sourceNode.metadata.height);
                    edge.metadata.tx = parseInt(edge.targetNode.metadata.x)+parseInt(edge.targetNode.metadata.width)/2;
                    edge.metadata.ty = parseInt(edge.targetNode.metadata.y);
                }else if(edge.sourceNode.metadata.depth>edge.targetNode.metadata.depth){
                    // Edges are top aligned from the source and bottom aligned from the target
                    edge.metadata.sx = parseInt(edge.sourceNode.metadata.x)+parseInt(edge.sourceNode.metadata.width)/2;
                    edge.metadata.sy = parseInt(edge.sourceNode.metadata.y);
                    edge.metadata.tx = parseInt(edge.targetNode.metadata.x)+parseInt(edge.targetNode.metadata.width)/2;
                    edge.metadata.ty = parseInt(edge.targetNode.metadata.y)+parseInt(edge.targetNode.metadata.height);
                }else if(parseInt(edge.sourceNode.metadata.x) < parseInt(edge.targetNode.metadata.x)){
                    // Edges are right aligned from the source and left aligned from the target
                    edge.metadata.sx = parseInt(edge.sourceNode.metadata.x)+parseInt(edge.sourceNode.metadata.width);
                    edge.metadata.sy = parseInt(edge.sourceNode.metadata.y)+parseInt(edge.sourceNode.metadata.height)/2;
                    edge.metadata.tx = parseInt(edge.targetNode.metadata.x);
                    edge.metadata.ty = parseInt(edge.targetNode.metadata.y)+parseInt(edge.targetNode.metadata.height)/2;
                }else{
                    // Edges are left aligned from the source and right aligned from the target
                    edge.metadata.sx = parseInt(edge.sourceNode.metadata.x);
                    edge.metadata.sy = parseInt(edge.sourceNode.metadata.y)+parseInt(edge.sourceNode.metadata.height)/2;
                    edge.metadata.tx = parseInt(edge.targetNode.metadata.x)+parseInt(edge.targetNode.metadata.width);
                    edge.metadata.ty = parseInt(edge.targetNode.metadata.y)+parseInt(edge.targetNode.metadata.height)/2;
                }
                // edge.metadata.lineColor = styleConfig.edgeLineColor;
                // Set edge line color to target node color
                if(!nodeList.metadata.matchBranchColors || !(edge.targetNode.metadata.domain!=null
                        && nodeList.branches[edge.targetNode.metadata.domain.toUpperCase()] != null)){
                    edge.metadata.lineColor = styleConfig.edgeStrokeColor[edge.metadata.type]
                        || styleConfig.edgeStrokeColor[edge.targetNode.uniqueType]
                        || styleConfig.edgeStrokeColor["default"];
                }else{
                    edge.metadata.lineColor = nodeList.branches[edge.targetNode.metadata.domain.toUpperCase()].color
                    || styleConfig.edgeStrokeColor[edge.metadata.type]
                        || styleConfig.edgeStrokeColor[edge.targetNode.uniqueType]
                        || styleConfig.edgeStrokeColor["default"];
                }
                edge.metadata.lineWidth = styleConfig.edgeLineWidth;
                edge.metadata.lineType = (!edge.targetNode.metadata.archive) ? "line" : "dashed";
            }
        }
        return nodeList
    }
    // Handle url seperation
    function splitDomain(url){
        const parts = url.split(".").reverse();
        return {
            "tld":parts[0].toUpperCase(),
            "domain":(parts.length>1) ? (`${parts[1]}.${parts[0]}`).toUpperCase() : null
        }
    }
    // Generates nodeList for a domain
    function nodeListFromDomain(domainInput, overrideMetadata={}, callback){
        const promise = new Promise(function(resolve,reject){
            // Use combination of PSL and built-in URL module
            // to accept a variety of input formats for the domain name,
            // ie. google.com. https://google.com, www.google.com
            try{
                // If fully qualified domain name use URL module 
                // to extract hostname
                const urlParsedDomain = new URL(domainInput);
                domainInput = urlParsedDomain.hostname;
            }catch(error){

            }
            const parsedDomainInput = splitDomain(domainInput);
            const domain = parsedDomainInput.domain;
            if(domain){
                const link = "/domains/"+domain;
                const rootNodeList = new NodeList(overrideMetadata);
                if(callback!=null){
                    if(typeof(callback)==='function'){
                        rootNodeList.updateFunction = callback;
                    }else{
                        throw Error("Invalid callback function");
                    }
                }else{
                    // Update callback defaults to empty function
                    rootNodeList.updateFunction = ()=>{};
                }
                // Load mapped links with zone data
                const mappedLinks = {};
                loadDNSCoffeeResponse("/zones",mappedLinks).then(()=>{
                    resolveNode(link, 0, mappedLinks, rootNodeList).then((nodeList)=>{
                        nodeList.updateLevels();
                        resolve(nodeList);
                    }).catch(error=>{
                        reject(error)
                    })
                }).catch((error)=>{
                    throw Error("Unable to resolve zones")
                });
            }else{
                reject(Error("Invalid Domain Name"));
            }
        })
        return promise;
    }
    // Uses D3.js to generate an svg representation of the nodelist
    function svgFromNodeList(nodeList,container,type="tree"){
        // Valid svg types
        const svgTypes = ["tree","mesh"];
        if(typeof(d3)==="undefined"){
            throw Error("Dependency d3.js not loaded")
        }else if(!(nodeList instanceof NodeList)){
            throw Error("Invalid parameter nodeList");
        }else if(!svgTypes.includes(type)){
            throw Error("Invalid parameter type for svg");
        }
        else{
            nodeList = formatNodeList(nodeList);
            const svgWidth = nodeList.metadata.width;
            const svgHeight= nodeList.metadata.height; 
            // Make copy of edge data to preserve original nodelist
            const edgeData = nodeList.edges.filter((edge)=>!edge.metadata.hidden)
            .map((edge)=>{
                return {
                    "id":edge.id,
                    "source":edge.source,
                    "sourceNode":edge.sourceNode,
                    "target":edge.target,
                    "targetNode":edge.targetNode,
                    "metadata":edge.metadata
                };
            });
            // Get container node
            const containerNode =d3.selectAll(container).node();
            // Create wrapper element within container
            const wrapper = d3.selectAll(container).append("div")
            .style('position','relative').style('width','100%')
            .style('max-height','inherit').style("z-index",0)
            .style('cursor','all-scroll')
            .style('overflow','hidden')
            .append("div")
            .style('position','relative').style('width','100%').style("z-index",0)
            .style('padding-bottom',((parseInt(svgHeight)/parseInt(svgWidth))*100)+"%")
            .style('vertical-align','top')
            .style('cursor','all-scroll')
            // Create div for key wrapper
            const keyWrapper = wrapper.append("div")
            .style("display","flex").style("justify-content","center")
            .style("align-items","center").style("background-color","rgba(0,0,0,0.3)")
            .style("z-index",-1).style("opacity",0)
            .style("width",containerNode.getBoundingClientRect().width+"px")
            .style("height",containerNode.getBoundingClientRect().height+"px")
            .style("position","absolute").style("top","0%").style("right","0%");
            // Create div for key
            const key = keyWrapper.append("div").style("background-color","#FFFFFF").style("position","absolute")
            .style("top","20px").style("bottom","20px").style("width","90%").style("display","flex")
            .style("align-items","center").style("justify-content","center").style("flex-direction","column")
            .style("border-radius","10px").style("box-shadow","0 0 5px rgba(0,0,0,0.5)").style("max-width","850px");
            // Create heading for domain
            key.append("div").style('font-weight',"bold").style("text-align","center")
            .style("margin","10px 0 0 5px").style("letter-spacing","1.5px").text("DOMAIN");
            // Create div for domain node and nameserver count
            key.append("div").style("position","relative").style("display","inline")
            .style("padding","5px 10px").style("border","solid 1px #000")
            .style("background-color","#75f581")
            .style("text-transform","uppercase").style("text-align","center")
            .style("font-size","16px").style("font-family","Monaco, monospace").text("EXAMPLE.COM")
            .append("div").style("width","24px").style("height","24px").style("border","solid 1px #000")
            .style("border-radius","50%").style("position","absolute").style("right","calc(-24px / 2")
            .style("top","calc(-24px / 2").style("background-color","#FFFFFF").style("text-align","center")
            .text("2").append("div").style("position","relative").style("top","-24px")
            .style("right","-24px").style("width","25vw")
            .style("text-align","left").style("font-size","12px")
            .style("font-family","sans-serif")
            .text("\u2190 Number of nameservers");;
            // Create heading for zone
            key.append("div").style('font-weight',"bold").style("text-align","center")
            .style("margin","20px 0 0 5px").style("letter-spacing","1.5px").text("ZONE");
            // Create div for zone node
            key.append("div").style("position","relative").style("display","inline")
            .style("padding","5px 10px").style("border","solid 1px #000")
            .style("background-color",nodeList.metadata.styleConfig.nodeFillColor.zone)
            .style("text-transform","uppercase").style("text-align","center")
            .style("font-size","16px").style("font-family","Monaco, monospace").text("COM");
            // Create heading for nameserver
            key.append("div").style('font-weight',"bold").style("text-align","center")
            .style("margin","20px 0 0 5px").style("letter-spacing","1.5px").text("NAMESERVER");
            // Create div for nameserver node and ip count and hazard symbol
            const nameserverKey = key.append("div").style("position","relative").style("display","inline")
            .style("padding","5px 10px").style("border","solid 1px #000")
            .style("background-color","#01d4ac")
            .style("text-transform","uppercase").style("text-align","center")
            .style("font-size","16px").style("font-family","Monaco, monospace").text("NS1.EXAMPLE.COM");
            nameserverKey.append("div").style("width","24px").style("height","24px").style("border","solid 1px #000")
            .style("border-radius","50%").style("position","absolute").style("right","calc(-24px / 2")
            .style("top","calc(-24px / 2").style("background-color","#FFFFFF").style("text-align","center")
            .text("?").append("div").style("position","relative").style("top","-24px")
            .style("right","-24px").style("width","25vw")
            .style("text-align","left").style("font-size","12px")
            .style("font-family","sans-serif")
            .text("\u2190 Domain not found in database");
             nameserverKey.append("div").style("width","0").style("height","0")
            .style("position","absolute").style("left","-18px").style("color","#FFF")
            .style("top","-34px").style("border","calc(27px * 0.666) solid transparent")
            .style("border-bottom","27px solid #000").style("text-align","center")
            nameserverKey.append("div").style("width","0").style("height","0")
            .style("position","absolute").style("left","-16px").style("color","#FFF")
            .style("top","-30px").style("border","calc(24px * 0.666) solid transparent")
            .style("border-bottom","24px solid #FF0000").style("text-align","center")
            .append("div").style("position","relative").style("left","-5px").style("top","2px").text("!")
            .append("div").style("position","relative").style("top","-24px")
            .style("left","calc(-4px - 25vw)").style("width","25vw")
            .style("text-align","right").style("font-size","12px")
            .style("font-family","sans-serif").style("color","#000")
            .text("Domain is hazardous \u2192");
            // Create heading for asn
            key.append("div").style('font-weight',"bold").style("text-align","center")
            .style("margin","20px 0 0 5px").style("letter-spacing","1.5px").text("AS (AUTONOMOUS SYSTEM)");
            // Create div for asns
            key.append("div").style("position","relative").style("display","inline")
            .style("padding","5px 10px").style("border","solid 1px #000")
            .style("background-color",nodeList.metadata.styleConfig.nodeFillColor.ip_accumulation)
            .style("text-transform","uppercase")
            .style("text-align","center").style("border-radius","100px")
            .style("font-size","16px").style("font-family","Monaco, monospace").text("AS:12345")
            .append("div").style("width","24px").style("height","24px").style("border","solid 1px #000")
            .style("border-radius","50%").style("position","absolute").style("right","calc(-24px / 2")
            .style("bottom","calc(-24px / 2").style("background-color","#FFFFFF").style("text-align","center")
            .text("5").append("div").style("position","relative").style("bottom","24px")
            .style("right","-24px").style("width","25vw")
            .style("text-align","left").style("font-size","12px")
            .style("font-family","sans-serif")
            .text("\u2190 Number of IPs");
            // Create div for key button
            const keyButton = wrapper.append("div")
            .style("display","flex").style("justify-content","center")
            .style("align-items","center").style("background-color","#00a9ff")
            .style("width","30px").style("height","30px")
            .style("position","absolute").style("top","20px").style("right","20px")
            .style("z-index",1000000).style("box-shadow","0 0 5px rgba(0,0,0,0.5)")
            .style("border-radius","50%").style("text-align","center")
            .style("font-weight","bold").style("font-family","Times,serif")
            .style("font-size","24px").style("color","#FFFFFF")
            .style("cursor","pointer").text("i");
            keyButton.on("mouseover",(d)=>{
                keyButton.transition().duration(150).ease(d3.easeLinear)
                .style("background-color","#FFFFFF").style("color","#00a9ff");
                keyWrapper.style("z-index",500000)
                .style("width",containerNode.getBoundingClientRect().width+"px")
                .style("height",containerNode.getBoundingClientRect().height+"px")
                keyWrapper.transition().duration(150).ease(d3.easeLinear)
                .style("opacity",1);
            }).on("mouseout",(d)=>{
                keyButton.transition().duration(150).ease(d3.easeLinear)
                .style("background-color","#00a9ff").style("color","#FFFFFF");
                keyWrapper.style("z-index",-1);
                keyWrapper.transition().duration(150).ease(d3.easeLinear)
                .style("opacity",0);
            })
            // Create div for tooltips
            // Make max-height 90% of frame height
            const tooltip =wrapper.append("div")
            .style("position","absolute").style("background-color","#FFFFFF")
            .style("padding","10px").style("z-index",-1)
            .style("opacity",0).style("box-shadow","0 0 5px rgba(0,0,0,0.5)")
            .style("border-radius","10px").style("overflow-y","auto")
            .style("max-height",(containerNode.getBoundingClientRect().height*0.9)+"px");
            // Create svg for graph
            const svg = wrapper.append("svg")
            .style('position','absolute')
            .style('top',0).style('left',0)
            .style('right',0).style('bottom',0)
            .attr('preserveAspectRatio','xMinYMin meet')
            .attr("viewBox", `0 0 ${svgWidth} ${svgHeight}`)
            .attr("version", 1.1)
            .attr("xmlns", "http://www.w3.org/2000/svg")
            .call(d3.zoom().scaleExtent([0.25, 5]).on("zoom", function () {
                svg.attr("transform", d3.event.transform)
                 
            })).append('g');
            
            // Create edges
            const edges = svg.selectAll("line").data(edgeData).enter()
            .append("line").style("stroke",(d)=>d.metadata.lineColor)
            .style("fill",(d)=>d.metadata.lineColor)
            .style("stroke-width",(d)=>d.metadata.lineWidth)
            // Add svgWidth/2 since nodes are center aligned and svg is left aligned
            .attr("x1",(d)=>parseInt(svgWidth/2)+parseInt(d.metadata.sx)).attr("y1",(d)=>d.metadata.sy)
            .attr("x2",(d)=>parseInt(svgWidth/2)+parseInt(d.metadata.tx)).attr("y2",(d)=>d.metadata.ty)
            .attr("stroke-dasharray",(d)=>(d.metadata.lineType=="dashed")?"4":"")
            .attr("marker-end",(d)=>{
                if(nodeList.metadata.matchBranchColors){
                    return `url(#endArrow_${d.targetNode.metadata.domain || d.metadata.type || d.targetNode.uniqueType})`;
                }else{
                    return `url(#endArrow_${d.metadata.type || d.targetNode.uniqueType})`;
                }
            })
            .attr("marker-start",
                (d)=>{
                    if(nodeList.metadata.matchBranchColors){
                        return (d.metadata.biDirectionalEdge)?`url(#startArrow_${d.targetNode.metadata.domain 
                            || d.metadata.type || d.targetNode.uniqueType})`:"";
                    }else{
                        return (d.metadata.biDirectionalEdge)?`url(#startArrow_${d.metadata.type 
                            || d.targetNode.uniqueType})`:"";
                    }
                    
                });
            // Create nodes (reverse to keep circles on top)
            const nodes = svg.selectAll("g").data(nodeList.levels.filter(level=>level!=null))
            .enter().append("g").selectAll("g").data((d,i,j)=>d.reverse()) /*Divide graph into groups by level*/
            .enter().append("g"); /*print each node*/
            // Create shape for each node
            nodes.append("rect")
            .attr("x", (d)=>parseInt(svgWidth/2)+parseInt(d.metadata.x))/*Add svgWidth/2 since nodes are center aligned and svg is left aligned*/
            .attr("y", (d)=>d.metadata.y)
            .style("fill",(d)=>d.metadata.fillColor)
            .style("stroke",(d)=>d.metadata.borderColor)
            .style("stroke-width",(d)=>d.metadata.borderWidth)
            .attr("width", (d)=>d.metadata.width)
            .attr("height", (d)=>d.metadata.height)
            .attr("rx", (d)=>d.metadata.borderRadius);
            // Create node text
            nodes.append("text")
            .style("font-size", (d)=>d.metadata.fontSize+"px")
            .style("font-family","Monaco, monospace")
            .attr("text-anchor", "middle")
            .attr("dominant-baseline", "middle")
            // Add svgWidth/2 since nodes are center aligned and svg is left aligned
            .attr("x", (d)=>parseInt(svgWidth/2)+parseInt(d.metadata.x)+ parseInt(d.metadata.width/2))
            .attr("y", (d)=>parseInt(d.metadata.y)+(d.metadata.height)/2)
            .attr("width", (d)=>d.metadata.width)
            .attr("height", (d)=>d.metadata.height).text((d)=>d.name)
            .attr("fill",(d)=>d.metadata.textColor);
            const hazardWrapper = nodes.filter((d)=>d.metadata.hazard || d.metadata.warning).append("g");
            // Add circle for hazard symbol
            hazardWrapper.append("polygon")
            .attr("points", (d)=>{
                const cx = parseInt(svgWidth/2)+parseInt(d.metadata.x);
                const cy = parseInt(d.metadata.y);
                const r = nodeList.metadata.styleConfig.nodeFontSize/2*1.5*1.333; /*Scale up by 33% to match tooltip*/
                return `${cx} ${cy-r+0.166*r}, ${cx+0.877*r} ${cy+.5*r+0.166*r}, ${cx+-0.877*r} ${cy+.5*r+0.166*r}`;
            })
            .style("fill",(d)=>(d.metadata.hazard) ? "#FF0000" : "#CA9E2A")
            .style("stroke","#000")
            .style("stroke-width",(d)=>d.metadata.borderWidth);
            hazardWrapper.append("text")
            .style("font-size", (d)=>d.metadata.fontSize+"px")
            .style("font-family","Monaco, monospace")
            .attr("text-anchor", "middle")
            .attr("dominant-baseline", "middle")
            .style("fill","#FFFFFF")
            .attr("x", (d)=>parseInt(svgWidth/2)+parseInt(d.metadata.x))
            .attr("y", (d)=>parseInt(d.metadata.y)+2)
            .attr("width", (d)=>d.metadata.width)
            .attr("height", (d)=>d.metadata.height)
            .text("!");
            // Add circle for tooltip nodes
            const nameserverTooltipWrapper = nodes.filter((d)=>(d.metadata.tooltip) 
                && (d.metadata.nameserverCount>0 || d.metadata.preload)).append("g");
            nameserverTooltipWrapper.append("circle")
            .attr("cx", (d)=>parseInt(svgWidth/2)+parseInt(d.metadata.x)
                +parseInt(d.metadata.tooltipOffset.nameserver.node.x))
            .attr("cy", (d)=>parseInt(d.metadata.y)+parseInt(d.metadata.tooltipOffset.nameserver.node.y))
            .attr("r", (d)=>nodeList.metadata.styleConfig.nodeFontSize/2*1.5) /*10% padding*/
            .style("fill","#FFFFFF")
            .style("stroke","#000")
            .style("stroke-width",(d)=>d.metadata.borderWidth)
            // Add count for hidden nodes
            nameserverTooltipWrapper.append("text")
            .style("font-size", (d)=>d.metadata.fontSize+"px")
            .style("font-family","Monaco, monospace")
            .attr("text-anchor", "middle")
            .attr("dominant-baseline", "middle")
            .style("fill","#000")
            .attr("x", (d)=>parseInt(svgWidth/2)+parseInt(d.metadata.x)
                +parseInt(d.metadata.tooltipOffset.nameserver.text.x))
            .attr("y", (d)=>parseInt(d.metadata.y)+parseInt(d.metadata.tooltipOffset.nameserver.text.y))
            .attr("width", (d)=>d.metadata.width)
            .attr("height", (d)=>d.metadata.height)
            .text(d=>(d.metadata.preload) ? "?" : d.metadata.nameserverCount);
            const ipTooltipWrapper = nodes.filter((d)=>(d.metadata.tooltip) && d.metadata.ipCount>0).append("g");
            ipTooltipWrapper.append("circle")
            .attr("cx", (d)=>parseInt(svgWidth/2)+parseInt(d.metadata.x)
                +parseInt(d.metadata.tooltipOffset.ip.node.x))
            .attr("cy", (d)=>parseInt(d.metadata.y)+parseInt(d.metadata.tooltipOffset.ip.node.y))
            .attr("r", (d)=>nodeList.metadata.styleConfig.nodeFontSize/2*1.5) /*10% padding*/
            .style("fill","#FFFFFF")
            .style("stroke","#000")
            .style("stroke-width",(d)=>d.metadata.borderWidth)
            // Add count for hidden nodes
            ipTooltipWrapper.append("text")
            .style("font-size", (d)=>d.metadata.fontSize+"px")
            .style("font-family","Monaco, monospace")
            .attr("text-anchor", "middle")
            .attr("dominant-baseline", "middle")
            .style("fill","#000")
            .attr("x", (d)=>parseInt(svgWidth/2)+parseInt(d.metadata.x)
                +parseInt(d.metadata.tooltipOffset.ip.text.x))
            .attr("y", (d)=>parseInt(d.metadata.y)+parseInt(d.metadata.tooltipOffset.ip.text.y))
            .attr("width", (d)=>d.metadata.width)
            .attr("height", (d)=>d.metadata.height)
            .text(d=>(d.metadata.preload) ? "?" : d.metadata.ipCount);
            function mouseover(d){
                // If tooltip node, use tooltip node for hover
                // Else use last tooltip node
                if(d.metadata!=null && d.metadata.tooltip!=null){
                    tooltip.html(d.metadata.tooltip);
                    tooltip.metadata = tooltip.metadata || {};
                    tooltip.metadata.lastNode=d;
                }else{
                    d=tooltip.metadata.lastNode;
                }
                tooltip.transition().duration(150)
                .ease(d3.easeLinear).style("opacity",1);
                // If node is not already being hovered on, reposition tooltip
                if(!d.metadata.mouseover){
                    // Unset width and height to let content take max width
                    tooltip.style("width","").style("height","").style("z-index",9999);
                    const tooltipPos = tooltip.node().getElementsByTagName("div")[0].getBoundingClientRect();
                    const tooltipWidth = tooltipPos.width+20; /*10px padding*/
                    const tooltipHeight = tooltipPos.height+20; /*10px padding*/
                    // Get scroll offset relative to document
                    const scrollTop = window.scrollY || document.documentElement.scrollTop;
                    const scrollLeft = window.scrollX || document.documentElement.scrollLeft;
                    // Get wrapper position to adjust offset
                    const wrapperPos = wrapper.node().getBoundingClientRect();
                    const wrapperTopOffset = wrapperPos.top+scrollTop;
                    const wrapperLeftOffset = wrapperPos.left+scrollLeft;
                    tooltip.style("width",tooltipWidth+"px")
                    .style("height",tooltipHeight+"px")
                    .style("max-height",(containerNode.getBoundingClientRect().height*0.9)+"px")
                    .style("top",(d3.event.pageY-wrapperTopOffset)+"px").style("left",(d3.event.pageX-wrapperLeftOffset)+"px");
                    d.metadata.mouseover=true;
                }
                d.metadata.mouseoverDebounce=true;
            }
            function mouseout(d){
                // If tooltip node, use tooltip node for hover
                // Else use last tooltip node
                if(d.metadata && d.metadata.tooltip){
                    tooltip.html(d.metadata.tooltip);
                    tooltip.metadata = tooltip.metadata || {};
                    tooltip.metadata.lastNode=d;
                }else{
                    d=tooltip.metadata.lastNode;
                }
                tooltip.transition().duration(150)
                .ease(d3.easeLinear).style("opacity",0);
                d.metadata.mouseoverDebounce=false;
                // Wait 150 milliseconds before hiding tooltip
                setTimeout(()=>{
                    if(!d.metadata.mouseoverDebounce){
                        d.metadata.mouseover=false;
                        tooltip.style("z-index",-1)
                    }
                },150);
            }
            // Add hoverbox for tooltip nodes
            nodes.filter((d)=>d.metadata.tooltip).selectAll("rect")
            .on("mouseover",mouseover).on("mouseout",mouseout);
            nodes.filter((d)=>d.metadata.tooltip).selectAll("text")
            .on("mouseover",mouseover).on("mouseout",mouseout);
            nodes.filter((d)=>d.metadata.tooltip).selectAll("circle")
            .on("mouseover",mouseover).on("mouseout",mouseout);
            // Maintain hover on tooltip
            tooltip.on("mouseover",()=>{mouseover(tooltip)}).on("mouseout",()=>{mouseout(tooltip)});
            // Create marker for arrows
            const edgeColors = {};
            Object.keys(nodeList.metadata.styleConfig.edgeStrokeColor).forEach((key)=>{
                edgeColors[key] = nodeList.metadata.styleConfig.edgeStrokeColor[key];
            })
            if(nodeList.metadata.matchBranchColors){
                Object.keys(nodeList.branches).forEach((branch)=>{
                    edgeColors[branch] = nodeList.branches[branch].color;
                })
            }
            const markerGroups = svg.append("defs").selectAll("g")
            .data(Object.keys(edgeColors)).enter().append("g");
            // Start Arrow
            markerGroups.append("marker").attr("id",(d)=>"startArrow_"+d)
            .attr("markerWidth",10).attr("markerHeight",7)
            .attr("refX",0).attr("refY",3.5).attr("orient","auto")
            .append("polygon").attr("points", "10 0, 0 3.5, 10 7")
            .attr("fill",(d)=>edgeColors[d]);
            // End arrow
            markerGroups.append("marker").attr("id",(d)=>"endArrow_"+d)
            .attr("markerWidth",10).attr("markerHeight",7)
            .attr("refX",10).attr("refY",3.5).attr("orient","auto")
            .append("polygon").attr("points", "0 0, 10 3.5, 0 7")
            .attr("fill",(d)=>edgeColors[d]);
            // Make nodes draggable in mesh mode
            if(type=="mesh"){
                function ticked(){
                    edges.attr("x1",d=>d.source.x).attr("y1",d=>d.source.y)
                    .attr("x2",d=>{return d.target.x}).attr("y2",d=>d.target.y);
                    nodes.attr("x",d=>d.x+6).attr("y",d=>d.y);
                    // Update text and rect posistions
                    // d.x, d.y are center coordinates for the rect, must be top-left based
                    nodes.selectAll("text")
                    .attr("x",d=>d.x)
                    .attr("y",d=>d.y)
                    nodes.selectAll("rect")
                    .attr("x",d=>d.x - parseInt(d.metadata.width)/2)
                    .attr("y",d=>d.y - parseInt(d.metadata.height)/2)
                    nodes.selectAll("circle")
                    .attr("cx",d=>d.x)
                    .attr("cy",d=>d.y) 
                    // Add circles and text for tooltip nodes
                    // Add circle for nameservers
                    nameserverTooltipWrapper.selectAll("circle")
                    .attr("cx", (d)=>d.x-parseInt(d.metadata.width)/2
                        +parseInt(d.metadata.tooltipOffset.nameserver.node.x))
                    .attr("cy", (d)=>d.y-parseInt(d.metadata.height)/2
                        +parseInt(d.metadata.tooltipOffset.nameserver.node.y));
                    // Add text for nameservers
                    nameserverTooltipWrapper.selectAll("text")
                    .attr("x", (d)=>d.x-parseInt(d.metadata.width)/2
                        +parseInt(d.metadata.tooltipOffset.nameserver.text.x))
                    .attr("y", (d)=>d.y-parseInt(d.metadata.height)/2
                        +parseInt(d.metadata.tooltipOffset.nameserver.text.y))
                    // Add cirle for IPs
                    ipTooltipWrapper.selectAll("circle")
                    .attr("cx", (d)=>d.x-parseInt(d.metadata.width)/2
                        +parseInt(d.metadata.tooltipOffset.ip.node.x))
                    .attr("cy", (d)=>d.y-parseInt(d.metadata.height)/2
                        +parseInt(d.metadata.tooltipOffset.ip.node.y));
                    // Add text for IPs
                    ipTooltipWrapper.selectAll("text")
                    .attr("x", (d)=>d.x-parseInt(d.metadata.width)/2
                        +parseInt(d.metadata.tooltipOffset.ip.text.x))
                    .attr("y", (d)=>d.y-parseInt(d.metadata.height)/2
                        +parseInt(d.metadata.tooltipOffset.ip.text.y))
                    // Add triangle for hazard nodes
                    hazardWrapper.selectAll("polygon") 
                    .attr("points", (d)=>{
                        const cx = d.x-parseInt(d.metadata.width)/2;
                        const cy = d.y-parseInt(d.metadata.height)/2;
                        const r = nodeList.metadata.styleConfig.nodeFontSize/2*1.5*1.333; /*Scale up by 33% to match tooltip*/
                        return `${cx} ${cy-r+0.166*r},
                                ${cx+0.877*r} ${cy+.5*r+0.166*r}, 
                                ${cx+-0.877*r} ${cy+.5*r+0.166*r}`;
                    });
                    // Add text for hazard nodes
                    hazardWrapper.selectAll("text")
                    .attr("x", (d)=>d.x-parseInt(d.metadata.width)/2)
                    .attr("y", (d)=>d.y-parseInt(d.metadata.height)/2+2)
                }
                function drag(simulation){
                    function dragstarted(d) {
                        if (!d3.event.active){
                            simulation.alphaTarget(0.3).restart();
                        }
                        d.fx = d.x;
                        d.fy = d.y;
                    }
                    function dragged(d) {
                        d.fx = d3.event.x;
                        d.fy = d3.event.y;
                    }
                    function dragended(d) {
                        if (!d3.event.active){ 
                            simulation.alphaTarget(0);
                        }
                        d.fx = null;
                        d.fy = null;
                    }
                    return d3.drag()
                        .on("start", dragstarted)
                        .on("drag", dragged)
                        .on("end", dragended);
                }
                // Start simulation
                const simulation = d3.forceSimulation(nodeList.nodes)
                .force("link",d3.forceLink().distance(150).id((d)=>d.id).links(edgeData))
                .force("charge",d3.forceManyBody().strength(-400))
                .force("center",d3.forceCenter(svgWidth/2,svgHeight/2))
                .force("collide",d3.forceCollide((d)=>d.metadata.width))
                .on("tick",ticked);
                nodes.call(drag(simulation));
            }
        }
    }
    // Exports
    exports.nodeListFromDomain = nodeListFromDomain;
    exports.graphMLStringFromNodeList = graphMLStringFromNodeList;
    exports.svgFromNodeList = svgFromNodeList;
})(DNSResolutionGrapher);