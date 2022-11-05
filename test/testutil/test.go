package main

import (

	"fmt"

)




func main(){

   nodeInfo:=[]int{1,1,2,3,4,6,6,6,6,7}
   fmt.Println(nodeInfo)
   for pos:=0 ;pos <len(nodeInfo);pos++{
      if nodeInfo[pos]==1{
         nodeInfo=append(nodeInfo[:pos], nodeInfo[pos+1:]...)
         pos--
      }
   }
   fmt.Println(nodeInfo)
}

