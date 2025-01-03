package server

type Tile struct {
	Resources map[string]int
	Players   []*Player
}

func CreateMap(width, height int) [][]*Tile {
	mapData := make([][]*Tile, height)
	for i := range mapData {
		mapData[i] = make([]*Tile, width)
		for j := range mapData[i] {
			mapData[i][j] = &Tile{
				Resources: map[string]int{"nourriture": 5},
				Players:   []*Player{},
			}
		}
	}
	return mapData
}
