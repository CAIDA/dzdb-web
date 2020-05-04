function PriorityQueue(compare){
    const list = [];
    const compareFunction = compare || ((a,b)=>a-b);
    const parentIndex = (index)=>(index>=1) ? Math.floor((index-1)/2):0;
    const parent = (index)=>list[parentIndex(index)] || null;
    const leftChildIndex = (index)=>2*index+1;
    const leftChild = (index)=>list[leftChildIndex(index)] || null;
    const rightChildIndex = (index)=>2*index+2;
    const rightChild = (index)=>list[rightChildIndex(index)] || null;
    const empty=()=>list.length==0;
    const peek=()=>((!empty()) ? list[0] : null);
    const swap=(index1,index2)=>{
        if(index1<0 || index1>=list.length || 
            index2<0 ||index2>=list.length){
            throw new Error("Index out of bounds");
        }else{
            const temp = list[index1];
            list[index1]=list[index2];
            list[index2]=temp;
        }
    }
    const add=function(n){
        list.push(n);
        let i = list.length-1;
        while(i != 0 && compareFunction(parent(i),list[i])>0){
            swap(parentIndex(i),i)
            i=parentIndex(i);
        }
    }
    const remove=function(){
        if(!empty()){
            const val = peek();
            const maxHeap = list.pop();
            // If not empty update strucutre
            if(!empty()){
                let currentIndex = 0;
                let positionFound = false;
                list[currentIndex] = maxHeap;
                while(leftChild(currentIndex)!=null && !positionFound){
                    const currentChildIndex = (rightChild(currentIndex)!=null && 
                        compareFunction(leftChild(currentIndex),rightChild(currentIndex))>0) 
                        ? rightChildIndex(currentIndex) : leftChildIndex(currentIndex);
                    if(compareFunction(list[currentIndex],list[currentChildIndex])>0){
                        swap(currentIndex,currentChildIndex);
                        currentIndex = currentChildIndex;   
                    }else{ 
                        positionFound=true;
                    }
                }
            }
            return val;
        }else{
            return null
        }
    }
    const size=()=>list.length;
    return {peek,empty,add,remove,size,list}
}