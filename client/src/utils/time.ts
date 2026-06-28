export const DAYS_OF_WEEK = [
    "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"
]

export const TIME_INTERVAL = 30

export const toTimeInputValue = (time: string): string => {
    if (!time) return "";
    const parts = time.split(":");
    if (parts.length < 2) return "";
    return `${parts[0].padStart(2, "0")}:${parts[1].padStart(2, "0")}`;
};

const generateTimeArray = (minInterval: number): string[] => {
    const array:string[] = []
    for (let i = 0; i <24 ; i++){
        const hour = i.toString().padStart(2, "0")
        for (let j = 0; j < (60/minInterval); j++){
            const minutes = (minInterval*j).toString().padStart(2, "0")
            array.push(hour+":"+minutes)
        }
    }
    return array;
}

const TIME_ARRAY = generateTimeArray(TIME_INTERVAL)

type OptionParam = {
    businessStartTime: string,
    businessEndTime: string
}

export const startTimeSlice = (endTime: string, options?: OptionParam): string[]=> {
    var startIndex = 0, endIndex: number;
    if(options === undefined)
        endIndex = TIME_ARRAY.indexOf(endTime)
    else if (endTime < options.businessEndTime){
        startIndex = TIME_ARRAY.indexOf(options.businessStartTime)
        endIndex = TIME_ARRAY.indexOf(endTime)
    }
    else{
        startIndex = TIME_ARRAY.indexOf(options.businessStartTime)
        endIndex = TIME_ARRAY.indexOf(options.businessEndTime) 
        if (endIndex === -1)
            return TIME_ARRAY
        endIndex +=1 
    }
    if (startIndex === -1 || endIndex === -1)
        return TIME_ARRAY
    return TIME_ARRAY.slice(startIndex, endIndex)
}

export const endTimeSlice = (startTime: string, options?: OptionParam): string[]=> {
    var startIndex: number, endIndex: number;
    if(options === undefined || startTime > options.businessStartTime)
        startIndex = TIME_ARRAY.indexOf(startTime) + 1
    else 
        startIndex = TIME_ARRAY.indexOf(options.businessStartTime)
    if (startIndex === -1) 
        return TIME_ARRAY
    else if(options){
        endIndex = TIME_ARRAY.indexOf(options.businessEndTime) 
        if (endIndex === -1) 
            return TIME_ARRAY
        return TIME_ARRAY.slice(startIndex, endIndex +1)
    }
    else
        return TIME_ARRAY.slice(startIndex)
}