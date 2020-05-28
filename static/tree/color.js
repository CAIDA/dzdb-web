const Color = {};
(function(exports){
    function contrastTextColor(color){
        let contrastWhite = 1.05 / (luminance(color) + 0.05);
        let contrastBlack = (luminance(color) + 0.05) / 0.05; 
        if(contrastWhite > contrastBlack){
            return "#FFFFFF";
        }else{
            return "#000000";
        }
    }
    function luminance(color){
        color = [...color]
        for(let i = 0; i < color.length; i++){
            color[i]/=255;
            if(color[i]<0.03928){
                color[i] /= 12.92;
            }else{
                color[i] = Math.pow((((color[i]) + 0.055) / 1.055), 2.4);
            }
        }
        let r = color[0];
        let g = color[1];
        let b = color[2];

        let l =  ((0.2126 * r) + (0.7152 * g) + (0.0722 * b));
        return l;
    }
    function rgbToHex(rgb){
        let hexArray = rgb.map((val)=>{
            let hexVal = val.toString(16);
            return hexVal.length == 1 ? "0" + hexVal : hexVal;
        })
        return "#"+hexArray.join('');
    }
    function random(){
        let randColor = [255,255,255].map((x)=>Math.round(x*Math.random()));
        return randColor;
    }
    exports.random = random;
    exports.rgbToHex = rgbToHex;
    exports.contrastTextColor = contrastTextColor;
})(Color);